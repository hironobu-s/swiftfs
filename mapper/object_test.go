package mapper

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"strings"

	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/openstack"
)

const (
	TEST_CONTAINER = "object-test-container"
	TEST_DIRECTORY = "test-directory"
	TEST_OBJECT    = "test-object"
	TEST_DATA      = "testdata"
)

var swift *openstack.Swift

func initSwift() {
	c := config.NewConfig()
	c.ContainerName = TEST_CONTAINER

	// initialize swift
	var err error
	swift = openstack.NewSwift(c)
	if err = swift.Auth(); err != nil {
		panic(err)
	}
	if err = swift.DeleteContainer(); err != nil {
		// 404
	}

	if err = swift.CreateContainer(); err != nil {
		panic(err)
	}
}

func TestLocalPath(t *testing.T) {
	path := TEST_OBJECT
	o := &object{
		Path: path,
	}

	if o.Localpath() != "/tmp/swiftfs/"+TEST_OBJECT {
		t.Fatalf("localpath mismatched %s != %s", path, o.Localpath())
	}
}

func TestDownload(t *testing.T) {
	var err error

	initSwift()

	// upload test object
	if err = swift.Upload(TEST_OBJECT, strings.NewReader(TEST_DATA)); err != nil {
		t.Fatalf("%v", err)
	}

	// download test
	path := TEST_OBJECT
	o := &object{
		Path:  path,
		swift: swift,
	}

	err = o.download()
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, err = os.Stat(o.Localpath())
	if err != nil {
		t.Fatalf("%v", err)
	}

	data, err := ioutil.ReadFile(o.Localpath())
	if err != nil {
		t.Fatalf("%v", err)
	} else if string(data) != TEST_DATA {
		t.Fatalf("File data does not match TEST_DATA")
	}
}

func TestOpen(t *testing.T) {
	path := TEST_OBJECT
	o := &object{
		Path: path,
	}

	file, err := o.Open(os.O_RDWR, 0600)
	if err != nil {
		t.Errorf("%v", err)

	} else if file.Name() != o.Localpath() {
		t.Errorf("localpath mismatched %s != %s", path, o.Localpath())
	}
}

func TestFlush(t *testing.T) {
	var err error
	path := TEST_OBJECT
	o := &object{
		Path: path,
	}

	loc, _ := time.LoadLocation("Asia/Tokyo")
	tt := time.Date(2015, 1, 1, 13, 0, 0, 0, loc)
	if err = os.Chtimes(o.Localpath(), tt, tt); err != nil {
		t.Fatalf("%v", err)
	}

	// o.Mtime will syncronize with mtime of localpath
	o.Flush()

	if !tt.Equal(o.Mtime) {
		t.Errorf("mtime mismatched %s != %s", tt.String(), o.Mtime.String())
	}
}

func TestUpload(t *testing.T) {
	var err error

	path := TEST_OBJECT + "2"
	o := &object{
		Path:  path,
		swift: swift,
	}

	file, err := o.Open(os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer file.Close()

	if _, err = file.WriteString(TEST_DATA); err != nil {
		t.Fatalf("%v", err)
	}

	if err = o.Upload(); err != nil {
		t.Fatalf("%v", err)
	}

	result := swift.Get(path)
	data, _ := ioutil.ReadAll(result.Body)
	if string(data) != TEST_DATA {
		t.Fatalf("File data does not match TEST_DATA")
	}
}
