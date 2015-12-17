package fs

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/mapper"
	"github.com/hironobu-s/swiftfs/openstack"
)

// --------------- utility funcs ---------------

func getCurrentUser() fuse.Owner {
	owner := fuse.Owner{
		Uid: 0,
		Gid: 0,
	}

	currentUser, err := user.Current()
	if err != nil {
		return owner
	}

	uid, err := strconv.ParseUint(currentUser.Uid, 10, 32)
	if err != nil {
		return owner
	}

	gid, err := strconv.ParseUint(currentUser.Gid, 10, 32)
	if err != nil {
		return owner
	}

	owner.Uid = uint32(uid)
	owner.Gid = uint32(gid)
	return owner
}

func getContext() *fuse.Context {
	c := &fuse.Context{
		Owner: getCurrentUser(),
		Pid:   uint32(os.Getpid()),
	}
	return c
}

// --------------- mount/unmount for tests ---------------

var fs *objectFileSystem
var server *fuse.Server

func mount() error {
	var err error

	if fs != nil || server != nil {
		// already mounting
		return nil
	}

	// create mountpoint
	os.Mkdir(TEST_MOUNTPOINT, 0777)

	// config
	config := &config.Config{
		MountPoint:      TEST_MOUNTPOINT,
		ContainerName:   TEST_CONTAINER_NAME,
		CreateContainer: true,
		Debug:           true,
		NoDaemon:        true,
	}

	// swift
	swift := openstack.NewSwift(config)
	if err = swift.Auth(); err != nil {
		return err
	}
	swift.DeleteContainer()

	// mapper
	mapper, err := mapper.NewObjectMapper(config)
	if err != nil {
		return err
	}

	// initialize filesystem
	fs = NewObjectFileSystem(config, mapper)

	path := pathfs.NewPathNodeFs(fs, nil)
	con := nodefs.NewFileSystemConnector(path.Root(), &nodefs.Options{})

	opts := &fuse.MountOptions{
		Name:   "test-filesystem",
		FsName: "test-filesystem",
	}

	// create server and do mount with dedicated goroutine
	server, err = fuse.NewServer(con.RawFS(), TEST_MOUNTPOINT, opts)
	if err != nil {
		return err
	}

	go func() {
		server.Serve()
	}()

	server.WaitMount()

	return nil
}

// --------------- tests ---------------

func TestNewObjectFileSystem(t *testing.T) {
	config := &config.Config{
		MountPoint:      TEST_MOUNTPOINT,
		ContainerName:   TEST_CONTAINER_NAME,
		CreateContainer: true,
	}

	mapper, err := mapper.NewObjectMapper(config)
	if err != nil {
		t.Fatalf("%v", err)
	}

	f := NewObjectFileSystem(config, mapper)

	if f.containerName != TEST_CONTAINER_NAME {
		t.Errorf("ContainerName is different (%s != %s)", f.containerName, TEST_CONTAINER_NAME)
	}

	if !f.createContainer {
		t.Errorf("CreateContainer option is false")
	}
}

// Mount filesystem
func TestBeforeAll(t *testing.T) {
	mount()
}

func TestCreate(t *testing.T) {
	filename := "testfile"

	c := getContext()
	_, stat := fs.Create(filename, 0, 0600, c)
	if !stat.Ok() {
		t.Errorf("Create fail")
	}

	var err error
	_, err = os.Stat(filepath.Join(TEST_MOUNTPOINT, filename))
	if err != nil {
		t.Errorf("Create fail(stat() returns error %v", err)
	}
}

func TestGetAttrFile(t *testing.T) {
	var err error
	name := "getattr_test_file"
	path := filepath.Join(TEST_MOUNTPOINT, name)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		t.Errorf("Create fail (open() returns error) %v", err)
	}

	_, err = f.WriteString("testdata")
	if err != nil {
		t.Errorf("Create fail (WriteFile() returns error) %v", err)
	}
	f.Close()

	c := getContext()
	attr, stat := fs.GetAttr(name, c)
	if !stat.Ok() {
		t.Errorf("GetAttr fail")
	}

	if attr.Owner.Uid != c.Uid {
		t.Errorf("mismatched uid")
	}

	if int(attr.Size) != len("testdata") {
		t.Errorf("mismatched size %d, %d", int(attr.Size), len("testdata"))
	}

	if attr.Mode&^fuse.S_IFREG == 0 {
		t.Errorf("invalid mode")
	}
}

func TestGetAttrDir(t *testing.T) {
	var err error
	name := "getattr_test_dir"
	path := filepath.Join(TEST_MOUNTPOINT, name)

	if err = os.Mkdir(path, 0755); err != nil {
		t.Errorf("mkdir() fail(%v)", err)
	}

	c := getContext()
	attr, stat := fs.GetAttr(name, c)
	if !stat.Ok() {
		t.Errorf("GetAttr fail")
	}

	if attr.Owner.Uid != c.Uid {
		t.Errorf("mismatched uid")
	}

	if attr.Mode&^fuse.S_IFDIR == 0 {
		t.Errorf("invalid mode")
	}
}

func TestMkdir(t *testing.T) {
	//var err error
	name := "mkdir_test"
	path := filepath.Join(TEST_MOUNTPOINT, name)

	c := getContext()
	st := fs.Mkdir(path, 0755, c)
	if !st.Ok() {
		t.Errorf("Mkdir fail")
	}

	attr, st := fs.GetAttr(path, c)
	if !st.Ok() {
		t.Errorf("GetAttr fail(Mkdir)")
		return
	}

	if time.Now().Equal(attr.AccessTime()) {
		t.Errorf("Mismatched Time %s", path)
	}
}

func TestOpenDir(t *testing.T) {
	var err error
	name := "opendir_test"
	path := filepath.Join(TEST_MOUNTPOINT, name)
	c := getContext()

	stat := fs.Mkdir(path, 0775, c)
	if !stat.Ok() {
		t.Errorf("OpenDir fail (Mkdir() returns error) %v", err)
	}

	// testfile
	_, stat = fs.Create(filepath.Join(path, "testfile"), uint32(os.O_CREATE|os.O_RDWR), 0644, c)
	if !stat.Ok() {
		t.Errorf("OpenDir fail (Create() returns error) %v", err)
	}

	// test directory
	fs.Mkdir(filepath.Join(path, "testdir"), 0755, c)
	if !stat.Ok() {
		t.Errorf("OpenDir fail (Mkdir() returns error) %v", err)
	}

	entries, stat := fs.OpenDir(path, c)
	if !stat.Ok() {
		t.Errorf("OpenDir fail")
	}

	if len(entries) != 2 {
		t.Errorf("error length of entries [%v]", entries)
	}

	for _, e := range entries {
		if e.Name == "testfile" && e.Mode&fuse.S_IFREG == 0 {
			t.Errorf("invalid mode(%s)", e.Name)
		} else if e.Name == "testdir" && e.Mode&fuse.S_IFDIR == 0 {
			t.Errorf("invalid mode(%s)", e.Name)
		} else if e.Name != "testfile" && e.Name != "testdir" {
			t.Errorf("Not enough entry(%s)", e.Name)
		}
	}
}

func TestOpen(t *testing.T) {
	name := "test-open"
	path := filepath.Join(TEST_MOUNTPOINT, name)
	c := getContext()

	var stat fuse.Status
	_, stat = fs.Create(path, uint32(os.O_CREATE), 0644, c)
	if !stat.Ok() {
		t.Errorf("Open fail")
	}

	_, stat = fs.GetAttr(path, c)
	if !stat.Ok() {
		t.Errorf("Open fail(GetAttr)")
	}
}

func TestUnlink(t *testing.T) {
	name := "test-unlink"
	path := filepath.Join(TEST_MOUNTPOINT, name)
	c := getContext()

	var stat fuse.Status
	_, stat = fs.Create(path, uint32(os.O_CREATE), 0644, c)
	if !stat.Ok() {
		t.Errorf("Unlink fail(Create)")
	}

	_, stat = fs.GetAttr(path, c)
	if !stat.Ok() {
		t.Errorf("Unlink fail(GetAttr1)")
	}

	stat = fs.Unlink(path, c)
	if !stat.Ok() {
		t.Errorf("Unlink fail(GetAttr1)")
	}

	_, stat = fs.GetAttr(path, c)
	if stat != fuse.ENOENT {
		t.Errorf("Unlink fail(status is not ENOENT)")
	}
}

func TestStatFs(t *testing.T) {
	statfs := fs.StatFs("")
	if statfs == nil {
		t.Errorf("StatFS() returns nil")
	}
}

func TestLink(t *testing.T) {
	stat := fs.Link("from-name", "to-name", getContext())
	if stat != fuse.ENOSYS {
		t.Errorf("Link() should returns ENOSYS")
	}
}

func TestRename(t *testing.T) {
	oldname := "test-rename-old"
	newname := "test-rename-new"
	oldpath := filepath.Join(TEST_MOUNTPOINT, oldname)
	newpath := filepath.Join(TEST_MOUNTPOINT, newname)

	c := getContext()

	var stat fuse.Status
	_, stat = fs.Create(oldpath, uint32(os.O_CREATE), 0644, c)
	if !stat.Ok() {
		t.Errorf("Rename fail")
	}

	_, stat = fs.GetAttr(oldpath, c)
	if !stat.Ok() {
		t.Errorf("Rename fail(GetAttr for oldpath after rename)")
	}

	stat = fs.Rename(oldpath, newpath, c)
	if !stat.Ok() {
		t.Errorf("Rename fail(Rename)")
	}

	_, stat = fs.GetAttr(newpath, c)
	if !stat.Ok() { // newpath should be OK
		t.Errorf("Rename fail(GetAttr for newpath)")
	}

	_, stat = fs.GetAttr(oldpath, c)
	if stat != fuse.ENOENT { // oldpath should be ENOENT
		t.Errorf("Rename fail(GetAttr for oldpath")
	}
}

func TestRmdir(t *testing.T) {
	name := "rmdir_test"
	path := filepath.Join(TEST_MOUNTPOINT, name)

	c := getContext()
	st := fs.Mkdir(path, 0755, c)
	if !st.Ok() {
		t.Errorf("Rmdir fail(mkdir)")
	}

	_, st = fs.GetAttr(path, c)
	if !st.Ok() {
		t.Errorf("GetAttr fail(Rmdir)")
		return
	}

	st = fs.Rmdir(path, c)
	if !st.Ok() {
		t.Errorf("Rmdir fail")
	}

	_, st = fs.GetAttr(path, c)
	if st != fuse.ENOENT {
		t.Errorf("Rmdir (GetAttr should returns ENOENT)")
		return
	}
}

// Unmount after run all tests
func TestAfterAll(t *testing.T) {
	server.Unmount()
	os.RemoveAll(TEST_MOUNTPOINT)
}
