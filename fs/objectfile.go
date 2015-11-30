package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hironobu-s/swiftfs/openstack"
)

type ObjectFile struct {
	swift *openstack.Swift

	Inode  *nodefs.Inode
	name   string
	file   *os.File
	lock   sync.Mutex
	object *openstack.Object

	needUpload bool

	nodefs.File
}

func NewObjectFile(name string, swift *openstack.Swift, object *openstack.Object) (*ObjectFile, error) {
	filename := "objfs" + strings.Replace(name, "/", "-", -1)
	f, err := os.OpenFile(filepath.Join(os.TempDir(), filename), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	log.Debugf("NewObjectFile:%s tmpfile=%s", name, f.Name())
	o := &ObjectFile{
		Inode:      nil,
		name:       name,
		swift:      swift,
		file:       f,
		needUpload: false,
		object:     object,
	}

	if err := o.download(); err != nil {
		return nil, err
	}

	return o, nil
}

func (o *ObjectFile) download() error {
	obj, err := o.swift.Get(o.name)
	if err != nil {
		log.Warnf("Error downloading %s. %v", o.name, err)
		return err
	}
	defer obj.Body.Close()

	_, err = io.Copy(o.file, obj.Body)
	if err != nil {
		log.Warnf("Can't copy %s to tmp-file. %v", o.name, err)
		return err
	}

	return nil
}

func (o *ObjectFile) InnerFile() nodefs.File {
	return nil
}

func (o *ObjectFile) SetInode(n *nodefs.Inode) {
	log.Debugf("SetInode %s", o.name)
	o.Inode = n
}

func (o *ObjectFile) String() string {
	return fmt.Sprintf("ObjectFile")
}

func (o *ObjectFile) Read(buf []byte, off int64) (res fuse.ReadResult, code fuse.Status) {
	if off == 0 {
		log.Debugf("Read %s offset=%d bytes=%d", o.file.Name(), off, len(buf))
	}

	o.lock.Lock()
	res = fuse.ReadResultFd(o.file.Fd(), off, len(buf))
	o.lock.Unlock()

	return res, fuse.OK
}

func (o *ObjectFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	if off == 0 {
		log.Debugf("Write %s offset=%d, length=%d", o.file.Name(), off, len(data))
	}

	o.lock.Lock()
	o.needUpload = true
	n, err := o.file.WriteAt(data, off)
	o.lock.Unlock()

	return uint32(n), fuse.ToStatus(err)
}

func (o *ObjectFile) Release() {
	log.Debugf("Release %s", o.file.Name())

	o.lock.Lock()
	o.file.Close()
	os.Remove(o.file.Name())
	o.lock.Unlock()
}

func (o *ObjectFile) Flush() fuse.Status {
	log.Debugf("Flush  %s", o.file.Name())

	o.lock.Lock()
	var err error
	if o.needUpload {
		err = o.swift.Upload(o.name, o.file)
		o.needUpload = false
	}
	o.lock.Unlock()

	if err != nil {
		return fuse.ToStatus(err)
	}

	stat, err := os.Stat(o.file.Name())
	if err == nil {
		o.object.Size = uint64(stat.Size())
	}
	return fuse.ToStatus(err)
}

func (o *ObjectFile) Fsync(flags int) (code fuse.Status) {
	log.Debugf("Fsync %s", o.name)

	o.lock.Lock()
	r := fuse.ToStatus(syscall.Fsync(int(o.file.Fd())))
	o.swift.Upload(o.name, o.file)
	o.lock.Unlock()

	return r
}

func (o *ObjectFile) Truncate(size uint64) fuse.Status {
	log.Debugf("Truncate %s", o.name)

	o.lock.Lock()
	r := fuse.ToStatus(syscall.Ftruncate(int(o.file.Fd()), int64(size)))
	o.lock.Unlock()

	return r
}

func (o *ObjectFile) Chmod(mode uint32) fuse.Status {
	return fuse.OK
}

func (o *ObjectFile) Chown(uid uint32, gid uint32) fuse.Status {
	return fuse.OK
}

func (o *ObjectFile) GetAttr(a *fuse.Attr) fuse.Status {
	log.Debugf("GetAttr(obj) %s", o.file.Name())

	o.lock.Lock()
	st := syscall.Stat_t{}
	err := syscall.Fstat(int(o.file.Fd()), &st)
	o.lock.Unlock()
	if err != nil {
		return fuse.ToStatus(err)
	}
	a.FromStat(&st)

	return fuse.OK
}

func (o *ObjectFile) Allocate(off uint64, size uint64, mode uint32) (code fuse.Status) {
	return fuse.OK
}

func (o *ObjectFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	return fuse.OK
}
