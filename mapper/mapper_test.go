package mapper

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hironobu-s/swiftfs/config"
)

var mapper *ObjectMapper

func initMapper() {
	mapper.objects = map[string]*object{}
	swift.DeleteContainer()
	swift.CreateContainer()
}

func TestNewObjectMapper(t *testing.T) {
	// if we need debug messages, please comment it out.
	// logrus.SetLevel(logrus.DebugLevel)

	// init swift
	if err := initSwift(); err != nil {
		t.Fatalf("%v", err)
	}
	swift.DeleteContainer()
	swift.CreateContainer()

	// Upload test directory and test data before initialize mapper
	dirname := "test-directory"
	swift.Upload(TEST_OBJECT, strings.NewReader(TEST_DATA))
	swift.MakeDirectory(dirname)

	// init mapper
	c := config.NewConfig()
	c.ContainerName = TEST_CONTAINER
	mapper, _ = NewObjectMapper(c)

	// test to exist local file or directory by syncObject()
	obj, ok := mapper.objects[dirname]
	if !ok {
		t.Fatalf("Directory %s not found", dirname)
	} else if obj.Type != DIRECTORY {
		t.Fatalf("invalid object type %s.(not directory)", dirname)
	}

	obj, ok = mapper.objects[TEST_OBJECT]
	if !ok {
		t.Fatalf("Object %s not found", TEST_OBJECT)
	} else if obj.Type != FILE {
		t.Fatalf("invalid object type %s.(not file)", TEST_OBJECT)
	}
}

func TestStat(t *testing.T) {
	if _, err := mapper.Stat(); err != nil {
		t.Fatalf("%v", err)
	}
}

func TestGet(t *testing.T) {
	// Do not execute initiMapper()

	obj, ok := mapper.Get(TEST_OBJECT)
	if !ok || obj == nil {
		t.Fatalf("object %s not found", TEST_OBJECT)
	}
}

func TestCreate(t *testing.T) {
	var err error
	initMapper()

	objname := TEST_OBJECT + "-test-create"
	obj, err := mapper.Create(objname)
	if err != nil || obj == nil {
		t.Fatalf("%v", err)
	}

	// object exists on object storage?
	r := swift.Get(objname)
	if r.Body == nil {
		t.Fatalf("object %s was created but not found on object storage", objname)
	}
	defer r.Body.Close()

	swift.Delete(objname)
}

func TestRename(t *testing.T) {
	var err error
	initMapper()

	objfrom := TEST_OBJECT + "-from"
	objto := TEST_OBJECT + "-to"

	_, err = mapper.Create(objfrom)
	if err != nil {
		t.Fatalf("create error %s", err)
	}

	if err = mapper.Rename(objfrom, objto); err != nil {
		t.Fatalf("rename error %s", err)
	}

	// local file exists ?
	obj, _ := mapper.Get(objto)
	_, err = os.Stat(obj.Localpath())
	if err != nil {
		t.Fatalf("%v", err)
	}
}

func TestDelete(t *testing.T) {
	var err error
	initMapper()

	objname := TEST_OBJECT + "-test-delete"
	obj, err := mapper.Create(objname)
	if err != nil || obj == nil {
		t.Fatalf("%v", err)
	}

	file, err := obj.Open(os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("%v", err)
	}
	file.WriteString(TEST_DATA)
	file.Close()

	if err = mapper.Delete(objname); err != nil {
		t.Fatalf("%v", err)
	}

	_, err = os.Stat(file.Name())
	if err == nil {
		t.Fatalf("file %s sill exists. %v", objname, err)
	}
}

func TestMkdir(t *testing.T) {
	var err error
	initMapper()

	dir, err := mapper.Mkdir(TEST_DIRECTORY)
	if err != nil {
		t.Fatalf("%v", err)
	}

	r := swift.Get(dir.Name)
	if r.Body == nil {
		t.Fatalf("Directory was not created on object storage")
	}
}

func TestRmdir(t *testing.T) {
	var err error
	initMapper()

	mapper.Mkdir(TEST_DIRECTORY)

	objname := filepath.Join(TEST_DIRECTORY, TEST_OBJECT)
	obj, err := mapper.Create(objname)
	if err != nil {
		t.Fatalf("%v", err)
	}

	file, err := obj.Open(os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("%v", err)
	}
	file.WriteString(TEST_DATA)
	file.Close()

	if err = mapper.Rmdir(TEST_DIRECTORY); err != nil {
		t.Fatalf("%v", err)
	}

	_, ok := mapper.Get(objname)
	if ok {
		t.Fatalf("Object still exists in mapper")
	}

	// following requests should be 404 status.
	r := swift.Get(TEST_DIRECTORY)
	if r.Err == nil {
		t.Fatalf("Directory still exists on object storage")
	}
	r = swift.Get(objname)
	if r.Err == nil {
		t.Fatalf("Object still exists on object storage")
	}
}

func TestOpenDir(t *testing.T) {
	var err error
	initMapper()

	dir, err := mapper.Mkdir(TEST_DIRECTORY)
	if err != nil {
		t.Fatalf("%v", err)
	}

	i := 0
	num := 3
	names := make([]string, num)
	for i < num {
		names[i] = filepath.Join(TEST_DIRECTORY, TEST_OBJECT+strconv.Itoa(i))
		_, err = mapper.Create(names[i])
		if err != nil {
			t.Fatalf("%v", err)
		}
		i++
	}

	objects := mapper.OpenDir(dir.Name)
	if len(objects) != num {
		t.Fatalf("count of objects is not match %d != %d", len(objects), num)
	}
}
