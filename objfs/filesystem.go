package objfs

import (
	"fmt"
	"io/ioutil"
	"os"

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
	objects         []drivers.Object

	pathfs.FileSystem
}

func NewFileSystem(config *Config) *fileSystem {
	fs := &fileSystem{
		mountPoint:      config.MountPoint,
		driver:          config.Driver,
		containerName:   config.ContainerName,
		createContainer: config.CreateContainer,
		FileSystem:      pathfs.NewDefaultFileSystem(),
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
	fs.objects = fs.driver.List()
}

func (fs *fileSystem) findObject(name string) *drivers.Object {
	for _, obj := range fs.objects {
		if obj.Name == name {
			return &obj
		}
	}
	return nil
}

// ------------------------

func (fs *fileSystem) String() string {
	return "objfs"
}

func (fs *fileSystem) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	var attr *fuse.Attr
	if name == "" {

		log.Debugf("GetAttr: (root) and refreash object list.")

		fs.buildObjectList()

		attr = &fuse.Attr{
			Mode: fuse.S_IFDIR | 0755,
		}
		return attr, fuse.OK

	} else {

		obj := fs.findObject(name)

		if obj != nil {
			log.Debugf("GetAttr: %s", name)

			attr = &fuse.Attr{
				Mode:  fuse.S_IFREG | 0644,
				Size:  obj.Size,
				Mtime: uint64(obj.LastModified.Unix()),
			}

			return attr, fuse.OK

		} else {
			log.Debugf("GetAttr: %s(no entry)", name)
			return nil, fuse.ENOENT
		}
	}
}

func (fs *fileSystem) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	log.Debugf("OpenDir: %s", name)

	entries := make([]fuse.DirEntry, len(fs.objects))

	var i = 0
	for _, obj := range fs.objects {
		entries[i] = fuse.DirEntry{Name: obj.Name, Mode: fuse.S_IFREG}
		i++
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
	return fuse.OK
}

func (fs *fileSystem) Chown(name string, uid uint32, gid uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}

func (fs *fileSystem) Access(name string, mode uint32, context *fuse.Context) (code fuse.Status) {
	return fuse.OK
}
func (fs *fileSystem) GetXAttr(name string, attribute string, context *fuse.Context) (data []byte, code fuse.Status) {
	return []byte(""), fuse.OK
}

func (fs *fileSystem) SetXAttr(name string, attr string, data []byte, flags int, context *fuse.Context) fuse.Status {
	return fuse.OK
}

func (fs *fileSystem) StatFs(name string) *fuse.StatfsOut {
	container, err := fs.driver.GetContainer()

	if err == nil {
		return &fuse.StatfsOut{
			Blocks:  container.Quota,
			Bsize:   1,
			Bfree:   container.Quota - container.Used*10,
			Bavail:  container.Quota - container.Used*10,
			Files:   container.Count,
			Ffree:   0,
			Frsize:  0,
			NameLen: 0,
		}
	} else {
		return nil
	}
}
