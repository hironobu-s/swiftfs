package openstack

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hironobu-s/swiftfs/config"
)

var client *Swift

const (
	TEST_CONTAINER_NAME = "objfs-test"
	TEST_OBJECT_NAME    = "testobject"
	TEST_OBJECT_DATA    = "hogehoge"
	TEST_DIRECTORY      = "testdirectory"
)

func TestMain(m *testing.M) {
	var err error

	c := config.NewConfig()
	c.ContainerName = TEST_CONTAINER_NAME

	client = NewSwift(c)
	if err = client.Auth(); err != nil {
		panic(err)
	}

	client.CreateContainer()
	code := m.Run()
	client.DeleteContainer()

	defer os.Exit(code)
}

func TestUpload(t *testing.T) {

	testobj, err := ioutil.TempFile("", "objfs")
	if err != nil {
		t.Errorf("%v", err)
	}
	defer os.Remove(testobj.Name())
	defer testobj.Close()

	_, err = testobj.Write([]byte(TEST_OBJECT_DATA))
	if err != nil {
		t.Errorf("%v", err)
	}
	testobj.Seek(0, 0)

	if err = client.Upload(TEST_OBJECT_NAME, testobj); err != nil {
		t.Errorf("%v", err)
	}
}

func TestGet(t *testing.T) {
	client.CreateContainer()

	obj, err := client.Get(TEST_OBJECT_NAME)
	if err != nil {
		t.Errorf("%v", err)
	}
	defer obj.Body.Close()

	body, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		t.Errorf("%v", err)
	}

	if string(body) != TEST_OBJECT_DATA {
		t.Errorf("Invalid object data (It's different from uploaded).")
	}
}

func TestList(t *testing.T) {
	objects := client.List()

	exists := false
	for _, obj := range objects.List() {
		if obj.Name == TEST_OBJECT_NAME {
			exists = true
			break
		}
	}
	if !exists {
		t.Errorf("Object not found. (upload failed?)")
	}
}

func TestDelete(t *testing.T) {
	if err := client.Delete(TEST_OBJECT_NAME); err != nil {
		t.Errorf("%v", err)
	}

	objects := client.List()

	exists := false
	for _, obj := range objects.List() {
		if obj.Name == TEST_OBJECT_NAME {
			exists = true
			break
		}
	}
	if exists {
		t.Errorf("Delete Failed.")
	}
}

func TestDirectoryCreation(t *testing.T) {
	var err error

	err = client.MakeDirectory(TEST_DIRECTORY)
	if err != nil {
		t.Errorf("Directory creation failed %v", err)
	}

	_, err = client.Get(TEST_DIRECTORY)
	if err != nil {
		t.Errorf("Directory creation succeeded, but cann't found a directory on Swift")
	}

	err = client.RemoveDirectory(TEST_DIRECTORY)
	if err != nil {
		t.Errorf("Directory deletion failed")
	}

	dir, err := client.Get(TEST_DIRECTORY)
	if dir.Name == TEST_DIRECTORY {
		t.Errorf("Directory deletion succeeded, but a directory exists on Swift")
	}
}

func TestContainerCreation(t *testing.T) {
	var err error

	if err = client.CreateContainer(); err != nil {
		t.Errorf("%v", err)
	}

	_, err = client.GetContainer()
	if err != nil {
		t.Errorf("%v", err)
	}

	if err = client.DeleteContainer(); err != nil {
		t.Errorf("%v", err)
	}

	_, err = client.GetContainer()
	if err == nil {
		t.Errorf("Container deletion failed")
	}
}
