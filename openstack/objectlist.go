package openstack

import (
	"container/list"
	"io"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	FILE = iota
	DIRECTORY
)

type Container struct {
	Name  string
	Quota uint64
	Used  uint64
	Count uint64
}

type Object struct {
	Name         string
	Body         io.ReadCloser
	Size         uint64
	LastModified time.Time
	Type         int

	*list.Element
}

type ObjectList struct {
	objects map[string]*Object
}

func NewObjectList() *ObjectList {
	l := &ObjectList{
		objects: map[string]*Object{},
	}
	return l
}

func (l *ObjectList) List() map[string]*Object {
	return l.objects
}

func (l *ObjectList) Find(name string) *Object {
	obj, ok := l.objects[name]
	if ok {
		return obj
	} else {
		return nil
	}
}

func (l *ObjectList) Set(name string, size uint64, lastModified time.Time, objectType int) *Object {
	if objectType != DIRECTORY && objectType != FILE {
		log.Warnf("Undefined object type(%d)", objectType)
		return nil
	}

	obj := &Object{
		Name:         name,
		Body:         nil,
		Size:         size,
		LastModified: lastModified,
		Type:         objectType,
	}
	l.objects[name] = obj

	return obj
}

func (l *ObjectList) Delete(name string) {
	delete(l.objects, name)
}
