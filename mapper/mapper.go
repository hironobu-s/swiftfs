package mapper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/openstack"
)

const (
	FILE = iota
	DIRECTORY
)

type Object interface {
	Localpath() string
	Open(flag int, perm os.FileMode) (*os.File, error)
	Flush() error
	Upload() error
	download() error
}

type object struct {
	Path string // foo/bar/buz.txt
	Name string // buz.txt
	Dir  string // foo/bat
	Type int    // const FILE or DIRECTORY

	Size  uint64
	Mtime time.Time

	swift      *openstack.Swift
	downloaded bool
}

func (o *object) Localpath() string {
	p := strings.Replace(o.Path, "/", "-", -1)
	return filepath.Join(os.TempDir(), "swiftfs", p)
}

// Open Temporary file
// Need to call close() after useing.
func (o *object) Open(flag int, perm os.FileMode) (*os.File, error) {

	// Download the filedata from the object storage when filesystem try to open an localfile
	// But, it does not need to download if O_TRUNC or O_CREATE flag passed.
	_, err := os.Stat(o.Localpath())
	if (flag&os.O_TRUNC == 0 && flag&os.O_CREATE == 0) && err != nil {
		log.Debugf("Open temporary file %s with downloading flag:%d %v", o.Path, flag, err)
		if err := o.download(); err != nil {
			log.Warnf("Download error %s, %v", o.Path, err)
			return nil, err
		}
		o.downloaded = true

	} else {
		log.Debugf("Open temporary file %s flag:%d", o.Path, flag)
	}

	file, err := os.OpenFile(o.Localpath(), flag, perm)
	if err != nil {
		return nil, err
	}

	return file, err
}

func (o *object) download() error {
	// Do not use o.Open() method.
	file, err := os.OpenFile(o.Localpath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	result := o.swift.Get(o.Path)
	defer result.Body.Close()

	if _, err = io.Copy(file, result.Body); err != nil {
		return err
	}

	return nil
}

func (o *object) Flush() (err error) {
	// Do not use o.Open() method.
	file, err := os.OpenFile(o.Localpath(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// update stat
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	o.Size = uint64(stat.Size())
	o.Mtime = stat.ModTime()

	return nil
}

func (o *object) Upload() (err error) {
	// Do not use o.Open() method.
	file, err := os.OpenFile(o.Localpath(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Flush
	o.Flush()

	// upload to object storage
	return o.swift.Upload(o.Path, file)
}

func newObject(swift *openstack.Swift, path string, t int) (obj *object) {
	name := filepath.Base(path)
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	obj = &object{
		Path: path,
		Name: name,
		Dir:  dir,
		Type: t,

		Size:  0,
		Mtime: time.Now(),

		swift:      swift,
		downloaded: false,
	}
	return obj
}

// --------------------------------------

type ObjectMapper struct {
	tmproot string // tmp path
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
		tmproot: c.TempDirectory,
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
	defer log.Debugf("[mapper] Get %s ok=%v", path, ok)

	obj, ok = m.objects[path]
	return obj, ok
}

func (m *ObjectMapper) Create(path string) (obj *object, err error) {
	defer log.Debugf("[mapper] Create %s error=%v", path, err)

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
	defer log.Debugf("[mapper] Rename %s to %s, error=%v", oldPath, newPath, err)

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
	defer log.Debugf("[mapper] Delete %s error=%v", path, err)

	obj, ok := m.objects[path]
	if !ok {
		return fmt.Errorf("Object (%s) not found", path)
	}

	if err := m.swift.Delete(path); err != nil {
		return err
	}

	os.Remove(obj.Localpath())
	delete(m.objects, path)

	return nil
}

// ----- Directory operations
func (m *ObjectMapper) OpenDir(dirname string) []*object {
	defer log.Debugf("[mapper] OpenDir %s", dirname)

	list := make([]*object, 0, 100)
	for _, obj := range m.objects {
		if obj.Dir == dirname {
			log.Debugf("OpenDir() match %s", obj.Dir)
			list = append(list, obj)
		}
	}
	return list
}

func (m *ObjectMapper) Mkdir(path string) (obj *object, err error) {
	defer log.Debugf("[mapper] Mkdir  %s error=%v", path, err)

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
	defer log.Debugf("[mapper] Rmdir %s ", path)
	return m.Delete(path)
}
