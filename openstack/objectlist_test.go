package openstack

import (
	"testing"
	"time"
)

func TestNewObjectList(t *testing.T) {
	l := NewObjectList()
	if l == nil {
		t.Errorf("NewObjectList() returns nil")
	}
}

func TestSet(t *testing.T) {
	l := NewObjectList()

	name := "testobject"
	size := uint64(20)
	tn := time.Now()
	tt := FILE

	obj := l.Set(name, size, tn, tt)

	if obj.Name != name {
		t.Errorf("Name is different (%s != %s)", name, obj.Name)
	}
	if obj.Size != size {
		t.Errorf("Size is different (%d != %d)", size, obj.Size)
	}
	if obj.LastModified != tn {
		t.Errorf("Time is different")
	}
	if obj.Type != tt {
		t.Errorf("Type is different (%d != %d)", tt, obj.Type)
	}
}

func TestList(t *testing.T) {
	l := NewObjectList()

	l.Set("foo", 10, time.Now(), FILE)
	l.Set("bar", 10, time.Now(), FILE)
	l.Set("baz", 10, time.Now(), FILE)

	list := l.List()
	if len(list) != 3 {
		t.Errorf("List() returns invalid list. (len = %d)", len(list))
	}
}

func TestFind(t *testing.T) {
	l := NewObjectList()

	l.Set("foo", 10, time.Now(), FILE)
	obj1 := l.Set("bar", 10, time.Now(), FILE)
	l.Set("baz", 10, time.Now(), FILE)

	obj2 := l.Find("bar")
	if obj2 == nil || obj1.Name != obj2.Name {
		t.Errorf("Find() returns invalid object")
	}

	obj4 := l.Find("hoge")
	if obj4 != nil {
		t.Errorf("Find() returns invalid object")
	}
}

func TestDelete(t *testing.T) {
	l := NewObjectList()

	l.Set("foo", 10, time.Now(), FILE)
	l.Set("bar", 10, time.Now(), FILE)
	l.Set("baz", 10, time.Now(), FILE)

	l.Delete("bar")

	list := l.List()
	if len(list) != 2 {
		t.Errorf("Delete() called, but an object is not deleted")
	}

	obj := l.Find("bar")
	if obj != nil {
		t.Errorf("Delete() called, but an object is not deleted")
	}
}
