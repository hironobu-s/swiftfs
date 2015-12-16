package fs

import (
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/mapper"
	"github.com/hironobu-s/swiftfs/openstack"
)

const (
	TEST_MOUNTPOINT     = "swiftfs-testrun"
	TEST_CONTAINER_NAME = "swiftfs-test"
	TEST_OBJECT_NAME    = "testobject"
	TEST_DATA           = "SwiftFS is the file system to mount Swift OpenStack Object Storage via FUSE. This product targets for Unix flatforms.This product may works on some OpenStack environments. We are testing on the following platforms."
)

var objfile *ObjectFile

func TestSetup(t *testing.T) {
	var err error

	c := config.NewConfig()
	c.MountPoint = TEST_MOUNTPOINT
	c.ContainerName = TEST_CONTAINER_NAME
	c.CreateContainer = true
	c.Debug = true
	c.NoDaemon = true

	// initialize swift and uplaod testdata
	swift := openstack.NewSwift(c)
	if err = swift.Auth(); err != nil {
		t.Fatalf("%v", err)
	}

	swift.DeleteContainer()
	swift.CreateContainer()

	// mapper
	mp, err := mapper.NewObjectMapper(c)
	if err != nil {
		t.Fatalf("%v", err)
	}
	obj, err := mp.Create(TEST_OBJECT_NAME)
	if err != nil {
		t.Fatalf("%v", err)
	}
	file, err := obj.Open(os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, err = file.WriteString(TEST_DATA)
	if err != nil {
		t.Fatalf("%v", err)
	}
	file.Close()

	// initialize ObjectFile
	objfile = NewObjectFile(TEST_OBJECT_NAME, obj)
}

func TestNewObjectFile(t *testing.T) {
	if objfile.name == "" {
		t.Fatalf("objfile.name is nil")
	}
	if objfile.object == nil {
		t.Fatalf("objfile.obj is nil")
	}
}

func TestOpenLocalFile(t *testing.T) {
	err := objfile.OpenLocalFile(uint32(os.O_RDWR), 0600)
	if err != nil {
		t.Fatalf("OpenLocalFile() fail %v", err)
	}

	stat, err := os.Stat(objfile.localfile.Name())
	if err != nil {
		t.Fatalf("Stat() fail")
	}
	if stat.Size() != int64(len(TEST_DATA)) {
		t.Fatalf("NewObjectFile() called, ")
	}
}

func TestSetInode(t *testing.T) {
	objfile.inode = nil

	inode := &nodefs.Inode{}

	objfile.SetInode(inode)
	if objfile.inode == nil {
		t.Fatalf("Inode is nil")
	}
}

func TestRead(t *testing.T) {
	compare := func(offset int64, size int64) {
		l := size - offset

		buf := make([]byte, l)
		res, code := objfile.Read(buf, offset)
		if !code.Ok() {
			t.Fatalf("Read() error, (not OK)")
		}

		read, _ := res.Bytes(buf)
		data := string(read)

		if TEST_DATA[offset:offset+l] != data {
			t.Fatalf("Read() returns invalid data. (offset:%d, size:%d)", offset, size)
		}
	}

	var i = 0
	for i < 10 {
		size := rand.Int63n(int64(len(TEST_DATA)))
		offset := rand.Int63n(size - 1)
		compare(offset, size)
		i++
	}
}

func TestWrite(t *testing.T) {
	prefix := "testdata"
	_, status := objfile.Write([]byte(prefix), 0)
	if status != fuse.OK {
		t.Fatalf("Write() returns error, %v", status)
	}

	buf := make([]byte, 32)
	res, code := objfile.Read(buf, 0)
	if !code.Ok() {
		t.Fatalf("Write() error, not OK")
	}

	r, _ := res.Bytes(buf)
	read := string(r)

	if !strings.HasPrefix(read, "testdata") {
		t.Fatalf("Write() returns invalid data, %s", read)
	}
}
