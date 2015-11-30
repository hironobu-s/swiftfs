package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"math/rand"

	"strings"

	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/openstack"
)

const (
	TEST_MOUNTPOINT     = "swiftfs-testrun"
	TEST_CONTAINER_NAME = "swiftfs-test"
	TEST_OBJECT_NAME    = "testobject"
	TEST_DATA           = "SwiftFS is the file system to mount Swift OpenStack Object Storage via FUSE. This product targets for Unix flatforms.This product may works on some OpenStack environments. We are testing on the following platforms."
)

var objfile *ObjectFile

func TestMain(m *testing.M) {
	var err error

	config := &config.Config{
		MountPoint:      TEST_MOUNTPOINT,
		ContainerName:   TEST_CONTAINER_NAME,
		CreateContainer: true,
		Debug:           true,
		NoDaemon:        true,
	}

	// initialize swift and uplaod testdata
	swift := openstack.NewSwift(config)
	if err = swift.Auth(); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	f, err := ioutil.TempFile("", "objcefile-test")
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	defer f.Close()

	f.WriteString(TEST_DATA)
	f.Seek(0, os.SEEK_SET)
	swift.Upload(TEST_OBJECT_NAME, f)

	// initialize
	obj := &openstack.Object{
		Name:         TEST_OBJECT_NAME,
		Body:         nil,
		Size:         uint64(len(TEST_DATA)),
		LastModified: time.Now(),
		Type:         openstack.FILE,
	}

	objfile, err = NewObjectFile(TEST_OBJECT_NAME, swift, obj)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	fmt.Printf("%v", objfile)

	os.Exit(m.Run())
}

func TestNewObjectFile(t *testing.T) {
	if objfile.file == nil {
		t.Errorf("objfile.file is nil")
	}
	stat, err := os.Stat(objfile.file.Name())
	if err != nil {
		t.Errorf("Stat() fail")
	}
	if stat.Size() != int64(len(TEST_DATA)) {
		t.Errorf("NewObjectFile() called, ")
	}
}

func TestSetInode(t *testing.T) {
	objfile.Inode = nil

	inode := &nodefs.Inode{}

	objfile.SetInode(inode)
	if objfile.Inode == nil {
		t.Errorf("Inode is nil")
	}
}

func TestRead(t *testing.T) {
	compare := func(offset int64, size int64) {
		l := size - offset

		buf := make([]byte, l)
		res, code := objfile.Read(buf, offset)
		if !code.Ok() {
			t.Errorf("Read() error, (not OK)")
		}

		read, _ := res.Bytes(buf)
		data := string(read)

		if TEST_DATA[offset:offset+l] != data {
			t.Errorf("Read() returns invalid data. (offset:%d, size:%d)", offset, size)
		}
	}

	var i = 0
	for i < 10 {
		size := rand.Int63n(int64(len(TEST_DATA)))
		offset := rand.Int63n(size - 1)
		t.Logf("%d, %d", size, offset)
		compare(offset, size)
		i++
	}
}

func TestWrite(t *testing.T) {
	prefix := "testdata"
	objfile.Write([]byte(prefix), 0)

	buf := make([]byte, 32)
	res, code := objfile.Read(buf, 0)
	if !code.Ok() {
		t.Errorf("Write() error, (not OK)")
	}

	r, _ := res.Bytes(buf)
	read := string(r)

	if !strings.HasPrefix(read, "testdata") {
		t.Errorf("Write() returns the data that does not have prefix")
	}
}
