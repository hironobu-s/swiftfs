package fs

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hironobu-s/swiftfs/mapper"
)

type ObjectFile struct {
	name  string
	inode *nodefs.Inode

	object     mapper.Object
	localfile  *os.File
	needUpload bool

	//mapper *mapper.ObjectMapper
	lock sync.Mutex

	nodefs.File
}

func NewObjectFile(name string, obj mapper.Object) *ObjectFile {
	f := &ObjectFile{
		name:       name,
		object:     obj,
		lock:       sync.Mutex{},
		needUpload: false,

		File: nodefs.NewDefaultFile(),
	}

	return f
}

func (o *ObjectFile) OpenLocalFile(flag uint32, mode uint32) error {
	var err error
	o.localfile, err = o.object.Open(int(flag), os.FileMode(mode))
	if err != nil {
		return fmt.Errorf("[objectfile] Can't open file(%s) [%v]", o.name, err)
	}
	return nil
}

func (o *ObjectFile) SetInode(n *nodefs.Inode) {
	log.Debugf("[objectfile] SetInode %s", o.name)
	o.inode = n
}

func (o *ObjectFile) String() string {
	return fmt.Sprintf("ObjectFile")
}

func (o *ObjectFile) Read(buf []byte, off int64) (res fuse.ReadResult, code fuse.Status) {
	log.Debugf("[objectfile] Read %s offset=%d bytes=%d", o.name, off, len(buf))
	if off == 0 {
	}

	o.lock.Lock()
	res = fuse.ReadResultFd(o.localfile.Fd(), off, len(buf))
	o.lock.Unlock()

	return res, fuse.OK
}

func (o *ObjectFile) Write(data []byte, off int64) (uint32, fuse.Status) {
	log.Debugf("[objectfile] Write %s offset=%d, length=%d", o.name, off, len(data))
	if off == 0 {
	}

	o.lock.Lock()
	n, err := o.localfile.WriteAt(data, off)
	o.needUpload = true
	o.lock.Unlock()

	if err != nil {
		log.Warnf("[objectfile] Write() error %v", err)
	}

	return uint32(n), fuse.ToStatus(err)
}

func (o *ObjectFile) Release() {
	log.Debugf("[objectfile] Release %s", o.name)

	if o.localfile != nil && o.needUpload {
		o.lock.Lock()
		if err := o.object.Upload(); err != nil {
			log.Warnf("[objectfile] Upload() error %s %v", o.name, err)
		}

		o.localfile.Close()
		o.lock.Unlock()
	}
}

func (o *ObjectFile) Flush() fuse.Status {
	log.Debugf("[objectfile] Flush  %s", o.name)

	if o.localfile != nil {
		if err := o.object.Flush(); err != nil {
			log.Warnf("[objectfile] Flush() error %s %v", o.name, err)
			return fuse.ToStatus(err)
		}
	}

	return fuse.OK
}

func (o *ObjectFile) Fsync(flags int) (code fuse.Status) {
	log.Debugf("[objectfile] Fsync %s", o.name)

	o.lock.Lock()
	r := fuse.ToStatus(syscall.Fsync(int(o.localfile.Fd())))
	o.lock.Unlock()

	return r
}

func (o *ObjectFile) Truncate(size uint64) fuse.Status {
	log.Debugf("[objectfile] Truncate %s", o.name)

	o.lock.Lock()
	r := fuse.ToStatus(syscall.Ftruncate(int(o.localfile.Fd()), int64(size)))
	o.needUpload = true
	o.lock.Unlock()

	return r
}

func (o *ObjectFile) Chmod(mode uint32) fuse.Status {
	if err := os.Chmod(o.localfile.Name(), os.FileMode(mode)); err != nil {
		return fuse.ToStatus(err)
	} else {
		return fuse.OK
	}
}

func (o *ObjectFile) Chown(uid uint32, gid uint32) fuse.Status {
	if err := os.Chown(o.localfile.Name(), int(uid), int(gid)); err != nil {
		return fuse.ToStatus(err)
	} else {
		return fuse.OK
	}
}

func (o *ObjectFile) GetAttr(a *fuse.Attr) fuse.Status {
	log.Debugf("[objectfile] GetAttr(obj) %s", o.name)

	stat, err := o.localfile.Stat()
	if err != nil {
		return fuse.EIO
	}
	st, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return fuse.ENODEV
	}
	a.FromStat(st)

	return fuse.OK
}

func (o *ObjectFile) Allocate(off uint64, size uint64, mode uint32) (code fuse.Status) {
	return fuse.OK
}

func (o *ObjectFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	return fuse.OK
}
