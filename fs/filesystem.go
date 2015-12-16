package fs

import (
	"os/user"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/mapper"
)

const (
	BLCOK_SIZE = 512
)

type objectFileSystem struct {
	containerName   string
	createContainer bool

	mapper *mapper.ObjectMapper

	lock sync.Mutex

	pathfs.FileSystem
}

func NewObjectFileSystem(c *config.Config, mapper *mapper.ObjectMapper) *objectFileSystem {
	fs := &objectFileSystem{
		containerName:   c.ContainerName,
		createContainer: c.CreateContainer,
		mapper:          mapper,
		lock:            sync.Mutex{},

		FileSystem: pathfs.NewDefaultFileSystem(),
	}
	return fs
}

// ------------------------

func (fs *objectFileSystem) String() string {
	return "swiftfs"
}

func (fs *objectFileSystem) getCurrentUser() fuse.Owner {
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

func (fs *objectFileSystem) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	var attr *fuse.Attr
	var owner = fs.getCurrentUser()

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if name == "" {
		//log.Debugf("GetAttr: (root)")

		attr = &fuse.Attr{
			Owner: owner,
			Mode:  fuse.S_IFDIR | 0755,
			Size:  4096,
		}
		return attr, fuse.OK
	}

	obj, ok := fs.mapper.Get(name)
	if !ok {
		//log.Debugf("GetAttr: %s(no entry)", name)
		return nil, fuse.ENOENT
	}

	switch obj.Type {
	case mapper.FILE:
		log.Debugf("GetAttr: %s(File) size:%d", obj.Name, obj.Size)

		attr = &fuse.Attr{
			Owner:  owner,
			Mode:   fuse.S_IFREG | 0644,
			Size:   obj.Size,
			Blocks: obj.Size / BLCOK_SIZE,
			Mtime:  uint64(obj.Mtime.Unix()),
		}

	case mapper.DIRECTORY:
		log.Debugf("GetAttr: %s(Directory) size:%d", obj.Name, obj.Size)
		attr = &fuse.Attr{
			Owner:  owner,
			Mode:   fuse.S_IFDIR | 0755,
			Size:   obj.Size,
			Blocks: obj.Size / BLCOK_SIZE,
			Mtime:  uint64(obj.Mtime.Unix()),
		}
	}
	return attr, fuse.OK
}

func (fs *objectFileSystem) OpenDir(dirname string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	log.Debugf("OpenDir: %s", dirname)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	entries := make([]fuse.DirEntry, 0, 1000)
	for _, obj := range fs.mapper.OpenDir(dirname) {
		log.Debugf("append dir entry: %s", obj.Path)

		var mode uint32
		switch obj.Type {
		case mapper.DIRECTORY:
			mode = fuse.S_IFDIR
		case mapper.FILE:
			mode = fuse.S_IFREG
		default:
			continue
		}
		entries = append(entries, fuse.DirEntry{Name: obj.Name, Mode: mode})
	}

	return entries, fuse.OK
}

func (fs *objectFileSystem) Create(name string, flags uint32, mode uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	log.Debugf("Create: %s, flags: %d", name, flags)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	// Add to mapper
	obj, err := fs.mapper.Create(name)
	if err != nil {
		log.Warnf("Can't append to mapper %v", err)
		return nodefs.NewDefaultFile(), fuse.EIO
	}

	file := NewObjectFile(name, obj)
	if err := file.OpenLocalFile(flags, mode); err != nil {
		log.Warnf("Create: OpenLocalFile() error %v", err)
		return file, fuse.EIO
	}

	return file, fuse.OK
}

func (fs *objectFileSystem) Open(name string, flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	log.Debugf("Open: %s, flags: %d", name, flags)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	obj, ok := fs.mapper.Get(name)
	if !ok {
		log.Warnf("Open: %s(no entry)", name)
		return nil, fuse.ENOENT

	} else if obj.Type == mapper.DIRECTORY {
		log.Warnf("Open: %s(DIRECTORY detected)", name)
		return nil, fuse.ENOENT
	}

	file := NewObjectFile(name, obj)
	if err := file.OpenLocalFile(flags, 0); err != nil {
		log.Warnf("Open() error %v", err)
		return file, fuse.EIO
	}

	return file, fuse.OK
}

func (fs *objectFileSystem) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Unlink: %s", name)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if fs.mapper.Delete(name) == nil {
		return fuse.OK
	} else {
		log.Warnf("Unlink fail(): %s", name)
		return fuse.ENOENT
	}
}

func (fs *objectFileSystem) Chmod(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Chmod %s", name)
	return fuse.OK
}

func (fs *objectFileSystem) Chown(name string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Chown %s", name)
	return fuse.OK
}

func (fs *objectFileSystem) StatFs(name string) *fuse.StatfsOut {
	container, err := fs.mapper.Stat()

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

func (fs *objectFileSystem) Link(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Link %s", oldName)
	return fuse.ENOSYS
}

func (fs *objectFileSystem) Mkdir(name string, mode uint32, context *fuse.Context) fuse.Status {
	log.Debugf("Mkdir %s", name)

	fs.lock.Lock()
	defer fs.lock.Unlock()
	_, err := fs.mapper.Mkdir(name)
	if err != nil {
		log.Debugf("Mkdir fail() %s %v", name, err)
		return fuse.EIO
	}
	return fuse.OK
}

func (fs *objectFileSystem) Rename(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Rename from %s to %s", oldName, newName)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	err := fs.mapper.Rename(oldName, newName)
	if err != nil {
		log.Debugf("Rename fail() %s %s %v", oldName, newName, err)
		return fuse.EIO
	}

	return fuse.OK
}

func (fs *objectFileSystem) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	log.Debugf("Rmdir %s", name)

	fs.lock.Lock()
	defer fs.lock.Unlock()

	if fs.mapper.Rmdir(name) == nil {
		return fuse.OK
	} else {
		log.Warnf("Rmdir() fail %s", name)
		return fuse.EIO
	}
}

func (fs *objectFileSystem) Utimens(name string, Atime *time.Time, Mtime *time.Time, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}
