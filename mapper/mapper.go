package mapper

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/openstack"
)

type ObjectMapper struct {
	objects map[string]*object
	swift   *openstack.Swift
}

func NewObjectMapper(c *config.Config) (*ObjectMapper, error) {
	var err error

	swift := openstack.NewSwift(c)
	if err = swift.Auth(); err != nil {
		return nil, err
	}

	if c.CreateContainer {
		if err = swift.CreateContainer(); err != nil {
			return nil, err
		}

	} else {
		_, err := swift.GetContainer()
		if err != nil {
			return nil, fmt.Errorf("Container \"%s\" not found", c.ContainerName)
		}
	}

	m := &ObjectMapper{
		objects: map[string]*object{},
		swift:   swift,
	}

	m.syncObjects()

	return m, nil
}

// ----- Sync between local and object storage
func (m *ObjectMapper) syncObjects() error {
	log.Debugf("syncObject() begin")
	objch, n := m.swift.List()

N:
	for {
		select {
		case s := <-objch:
			var t int
			if s.ContentType == "application/directory" {
				t = DIRECTORY
			} else {
				t = FILE
			}

			log.Debugf("[mapper] syncObject() append %s %s", s.Name, s.ContentType)

			obj := newObject(m.swift, s.Name, t)
			obj.Size = uint64(s.Bytes)

			// gophercloudがタイムゾーンを考慮しないで返してくるっぽい？
			lm, err := time.Parse(time.RFC3339, s.LastModified+"Z")
			if err != nil {
				log.Debugf("Invalid time format[%s]", s.LastModified)
				lm = time.Now()
			}
			obj.Mtime = lm

			m.objects[s.Name] = obj

		case num := <-n:
			log.Debugf("syncObject() %d objects were appended", num)
			break N
		}
	}
	return nil
}

// ----- Stat operation
func (m *ObjectMapper) Stat() (openstack.Container, error) {
	return m.swift.GetContainer()
}

// ----- File operations
func (m *ObjectMapper) Get(path string) (obj *object, ok bool) {
	obj, ok = m.objects[path]
	log.Debugf("[mapper] Get %s ok=%v", path, ok)

	return obj, ok
}

func (m *ObjectMapper) Create(path string) (obj *object, err error) {
	log.Debugf("[mapper] Create %s", path)

	_, ok := m.objects[path]
	if ok {
		return nil, fmt.Errorf("Object already exists(localpath=%s)", path)
	}

	obj = newObject(m.swift, path, FILE)
	m.objects[path] = obj

	// upload to object storage
	if err = m.swift.Upload(path, strings.NewReader("")); err != nil {
		return nil, err
	}

	return obj, nil
}

func (m *ObjectMapper) Rename(oldPath string, newPath string) (err error) {
	log.Debugf("[mapper] Rename %s to %s", oldPath, newPath)

	obj, ok := m.objects[oldPath]
	if !ok {
		return fmt.Errorf("Object (%s) not found", oldPath)
	}

	// Copy localfile
	from, err := obj.Open(os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer from.Close()

	newobj := newObject(m.swift, newPath, obj.Type)
	to, err := newobj.Open(os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer to.Close()

	n, err := io.Copy(to, from)
	if err != nil {
		os.Remove(to.Name())
		return err
	}

	st, _ := os.Stat(newobj.Localpath())
	if st.Size() != n {
		os.Remove(to.Name())
		return fmt.Errorf("Copy() error")
	}

	// Flush new object
	if err = newobj.Flush(); err != nil {
		return err
	}

	// Append new object
	// Do not use Set() method. We should use Copy() method.
	m.objects[newPath] = newobj

	// Coping on object storage
	if err = m.swift.Copy(oldPath, newPath); err != nil {
		os.Remove(to.Name())
		return err
	}

	// Delete old object
	return m.Delete(oldPath)
}

func (m *ObjectMapper) Delete(path string) (err error) {
	log.Debugf("[mapper] Delete %s", path)

	obj, ok := m.objects[path]
	if !ok {
		return fmt.Errorf("Object (%s) not found", path)
	}

	if err := m.swift.Delete(path); err != nil {
		return err
	}

	// Directory does not have localpath.
	if obj.Type == FILE {
		if err := os.Remove(obj.Localpath()); err != nil {
			return err
		}
	} else {
		log.Debugf("[mapper] Not delete localfile of %s. because type is directory", path)
	}
	delete(m.objects, path)

	return nil
}

// ----- Directory operations
func (m *ObjectMapper) OpenDir(dirname string) []*object {
	log.Debugf("[mapper] OpenDir %s", dirname)

	i := 0
	list := make([]*object, 0, len(m.objects))
	for _, obj := range m.objects {
		if obj.Dir == dirname {
			log.Debugf("OpenDir() match %s", obj.Dir)
			list = append(list, obj)
			i++
		}
	}
	list = list[:i]

	return list
}

func (m *ObjectMapper) Mkdir(path string) (obj *object, err error) {
	log.Debugf("[mapper] Mkdir  %s", path)

	o, ok := m.objects[path]
	if ok {
		return o, fmt.Errorf("Object already exists(localpath=%s)", path)
	}

	obj = newObject(m.swift, path, DIRECTORY)
	m.objects[path] = obj

	if err = m.swift.MakeDirectory(path); err != nil {
		return nil, err
	}

	return obj, nil
}

func (m *ObjectMapper) Rmdir(path string) error {
	for _, obj := range m.OpenDir(path) {
		if obj.Type == DIRECTORY {
			if err := m.Rmdir(obj.Name); err != nil {
				return err
			}
		} else {
			p := filepath.Join(path, obj.Name)
			log.Debugf("[mapper] Rmdir %s ", p)
			if err := m.Delete(p); err != nil {
				return err
			}
		}
	}

	log.Debugf("[mapper] Rmdir %s ", path)
	return m.Delete(path)
}
