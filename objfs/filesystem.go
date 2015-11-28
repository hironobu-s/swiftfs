package objfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hironobu-s/objfs/drivers"
)

type fileSystem struct {
	driver          drivers.Driver
	mountPoint      string
	containerName   string
	createContainer bool
	objectList      drivers.ObjectList

	lock sync.Mutex

	pathfs.FileSystem
}

func NewFileSystem(config *Config) *fileSystem {
	fs := &fileSystem{
		mountPoint:      config.MountPoint,
		driver:          config.Driver,
		containerName:   config.ContainerName,
		createContainer: config.CreateContainer,
		lock:            sync.Mutex{},

		FileSystem: pathfs.NewDefaultFileSystem(),
	}
	return fs
}

func (fs *fileSystem) Mount() (server *fuse.Server, err error) {
	if err = fs.driver.Auth(); err != nil {
		return nil, err
	}

	if fs.createContainer {
		if err = fs.driver.CreateContainer(); err != nil {
			return nil, err
		}

	} else {
		_, err := fs.driver.GetContainer()
		if err != nil {
			return nil, fmt.Errorf("Container \"%s\" not found", fs.containerName)
		}
	}

	path := pathfs.NewPathNodeFs(fs, nil)
	con := nodefs.NewFileSystemConnector(path.Root(), &nodefs.Options{
		EntryTimeout:    time.Second,
		AttrTimeout:     time.Second,
		NegativeTimeout: time.Second,
	})

	opts := &fuse.MountOptions{
		Name:   APP_NAME,
		FsName: APP_NAME,
	}

	server, err = fuse.NewServer(con.RawFS(), fs.mountPoint, opts)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func (fs *fileSystem) buildObjectList() {
	fs.objectList = fs.driver.List()
}

// ------------------------

func (fs *fileSystem) String() string {
	return "objfs"
}

func (fs *fileSystem) getCurrentUser() fuse.Owner {
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

func (fs *fileSystem) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	var attr *fuse.Attr
	var owner = fs.getCurrentUser()

	if name == "" {
		log.Debugf("GetAttr: (root)")

		attr = &fuse.Attr{
			Owner: owner,
			Mode:  fuse.S_IFDIR | 0755,
			Size:  4096,
		}
		return attr, fuse.OK
	}

	obj := fs.objectList.Find(name)
	if obj != nil {
		if obj.Type == drivers.DIRECTORY {
			log.Debugf("GetAttr: %s(directory)", name)
			attr = &fuse.Attr{
				Owner: owner,
				Mode:  fuse.S_IFDIR | 0755,
				Size:  4096,
				Nlink: 0,
			}

		} else {
			log.Debugf("GetAttr: %s", name)
			attr = &fuse.Attr{
				Owner: owner,
				Mode:  fuse.S_IFREG | 0644,
				Size:  obj.Size,
				Mtime: uint64(obj.LastModified.Unix()),
			}
		}
		return attr, fuse.OK

	} else {
		log.Debugf("GetAttr: %s(no entry)", name)
		return nil, fuse.ENOENT
	}
}

func (fs *fileSystem) OpenDir(dirname string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	log.Debugf("OpenDir: %s", dirname)

	fs.buildObjectList()

	entries := make([]fuse.DirEntry, 0, 1000)

	for _, obj := range fs.objectList {
		dir := filepath.Dir(obj.Name)
		if dir == "." {
			dir = ""
		}

		if dirname != dir {
			continue
		}
		log.Debugf("append dir entry: %s", filepath.Base(obj.Name))

		var mode uint32
		if obj.Type == drivers.DIRECTORY {
			mode = fuse.S_IFDIR
		} else {
			mode = fuse.S_IFREG
		}
		entries = append(entries, fuse.DirEntry{Name: filepath.Base(obj.Name), Mode: mode})

	}

	return entries, fuse.OK
}

func (fs *fileSystem) Create(name string, flags uint32, mode uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	log.Debugf("Create: %s, flags: %d", name, flags)

	var err error

	data, err := ioutil.TempFile("", "")
	if err != nil {
		log.Debugf("Temp Create Error: %v", err)
		return nil, fuse.ENOSYS
	}
	defer os.Remove(data.Name())
	defer data.Close()

	err = fs.driver.Upload(name, data)
	if err != nil {
		log.Debugf("Temp Create Error: %v", err)
		return nil, fuse.ENOSYS
	}

	fs.buildObjectList()

	file, err = NewObjectFile(name, fs.driver)
	if err != nil {
		log.Debugf("OBJECT ERROR: %v", err)
		return nil, fuse.ENOSYS
	}
	return file, fuse.OK
}

func (fs *fileSystem) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	log.Debugf("Open: %s, flags: %d", name, flags)

	file, err := NewObjectFile(name, fs.driver)
	if err != nil {
		log.Debugf("OBJECT ERROR: %v", err)
		return nil, fuse.ENOSYS
	}

	return file, fuse.OK
}

func (fs *fileSystem) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	err := fs.driver.Delete(name)
	if err != nil {
		log.Debugf("Delete Error: %v", err)
		return fuse.ENOSYS
	}

	fs.buildObjectList()

	return fuse.OK
}

func (fs *fileSystem) Chmod(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Chmod %s", name)
	return fuse.OK
}

func (fs *fileSystem) Chown(name string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Chown %s", name)
	return fuse.OK
}

func (fs *fileSystem) StatFs(name string) *fuse.StatfsOut {
	container, err := fs.driver.GetContainer()

	if err == nil {
		return &fuse.StatfsOut{
			Blocks:  container.Quota,
			Bsize:   1,
			Bfree:   container.Quota - container.Used,
			Bavail:  container.Quota - container.Used,
			Files:   container.Count,
			Ffree:   0,
			Frsize:  0,
			NameLen: 0,
		}
	} else {
		return nil
	}
}

func (fs *fileSystem) Link(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Link %s", oldName)
	return fuse.ENOSYS
}

func (fs *fileSystem) Mkdir(name string, mode uint32, context *fuse.Context) fuse.Status {
	log.Debugf("Mkdir %s", name)
	if err := fs.driver.MakeDirectory(name); err != nil {
		return fuse.ENOSYS

	} else {
		// refresh object list
		fs.buildObjectList()

		return fuse.OK
	}
}

func (fs *fileSystem) Rename(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Rename from %s to %s", oldName, newName)

	fs.lock.Lock()

	err := fs.driver.Copy(oldName, newName)
	if err != nil {
		log.Debugf("Copy Error: %v", err)
		return fuse.ENOSYS
	}

	err = fs.driver.Delete(oldName)
	if err != nil {
		log.Debugf("Delete Error: %v", err)

		fs.buildObjectList()
		return fuse.ENOSYS
	}

	fs.buildObjectList()

	fs.lock.Unlock()
	return fuse.OK
}

func (fs *fileSystem) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Rmdir %s", name)
	if err := fs.driver.RemoveDirectory(name); err != nil {
		return fuse.ENOSYS
	} else {
		fs.buildObjectList()
		return fuse.OK
	}
}
