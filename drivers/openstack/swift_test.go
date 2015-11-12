package openstack

import (
	"os"
	"testing"

	"io/ioutil"

	"github.com/hironobu-s/objfs/drivers/openstack"
)

var client *openstack.SwiftClient

const (
	TEST_CONTAINER_NAME = "objfs-test"
	TEST_OBJECT_NAME    = "testobject"
	TEST_OBJECT_DATA    = "hogehoge"
)

func TestMain(m *testing.M) {

	// flag.Parse()
	// fmt.Printf("%v", flag.Args())

	config := &openstack.SwiftConfig{
		ContainerName: TEST_CONTAINER_NAME,
	}

	client = openstack.NewSwiftClient(config)
	if err := client.Initialize(); err != nil {
		panic(err)
	}

	client.CreateContainer(TEST_CONTAINER_NAME)

	code := m.Run()
	defer os.Exit(code)

	client.DeleteContainer(TEST_CONTAINER_NAME)
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
	for _, obj := range objects {
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
	for _, obj := range objects {
		if obj.Name == TEST_OBJECT_NAME {
			exists = true
			break
		}
	}
	if exists {
		t.Errorf("Delete Failed.")
	}
}
