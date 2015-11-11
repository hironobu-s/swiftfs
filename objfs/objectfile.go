package objfs

import (
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hironobu-s/objfs/drivers"
)

type ObjectFile struct {
	name   string
	driver drivers.Driver
	file   *os.File
	lock   sync.Mutex

	needDownload bool
	needUpload   bool

	nodefs.File
}

func NewObjectFile(name string, driver drivers.Driver) (*ObjectFile, error) {

	//tmpfile := path.Join(os.TempDir(), "swiftfs-"+name)
	//f, err := os.OpenFile(tmpfile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	f, err := ioutil.TempFile("", "objfs-"+name)
	if err != nil {
		return nil, err
	}

	log.Debugf("NewObject:%s", name)

	o := &ObjectFile{
		name:         name,
		driver:       driver,
		file:         f,
		needDownload: true,
		needUpload:   false,
	}

	return o, nil
}

func (o *ObjectFile) InnerFile() nodefs.File {
	return nil
}

func (o *ObjectFile) SetInode(n *nodefs.Inode) {
	log.Debugf("SetInode %s", o.name)
}

func (o *ObjectFile) String() string {
	return fmt.Sprintf("object file")
}

func (o *ObjectFile) Read(buf []byte, off int64) (res fuse.ReadResult, code fuse.Status) {

	o.lock.Lock()
	defer o.lock.Unlock()

	// log.Debugf("Read %s offset=%d bytes=%d", o.name, off, length)
	if off == 0 {
		log.Debugf("Read request named %s.", o.name)
	}

	if o.needDownload {

		log.Debugf("Download %s from the object storage.", o.name)

		obj, err := o.driver.Get(o.name)
		if err != nil {
			log.Warnf("Can't get object[%s]: %v", o.name, err)
			return nil, fuse.EIO
		}
		defer obj.Body.Close()

		_, err = io.Copy(o.file, obj.Body)
		if err != nil {
			log.Warnf("Copy error: %v", err)
			return nil, fuse.EIO
		}

		o.needDownload = false
	}

	res = fuse.ReadResultFd(o.file.Fd(), off, len(buf))

	return res, fuse.OK
}

func (o *ObjectFile) Write(data []byte, off int64) (uint32, fuse.Status) {

	//log.Debugf("Write %s offset=%d, length=%d", o.name, off, len(data))
	if off == 0 {
		log.Debugf("Write request named %s.", o.name)
	}

	o.lock.Lock()
	defer o.lock.Unlock()

	o.needUpload = true

	n, err := o.file.WriteAt(data, off)
	return uint32(n), fuse.ToStatus(err)
}

func (o *ObjectFile) Release() {
	log.Debugf("Release %s", o.file.Name())

	o.lock.Lock()
	defer o.lock.Unlock()

	o.file.Close()

	os.Remove(o.file.Name())
}

func (o *ObjectFile) Flush() fuse.Status {

	log.Debugf("Flush  %s", o.name)

	o.lock.Lock()
	defer o.lock.Unlock()

	// Since Flush() may be called for each dup'd fd, we don't
	// want to really close the file, we just want to flush. This
	// is achieved by closing a dup'd fd.
	newFd, err := syscall.Dup(int(o.file.Fd()))

	if err != nil {
		return fuse.ToStatus(err)
	}
	err = syscall.Close(newFd)

	if o.needUpload {
		o.driver.Upload(o.name, o.file)
		o.needUpload = false
	}

	return fuse.ToStatus(err)
}

func (o *ObjectFile) Fsync(flags int) (code fuse.Status) {
	log.Debugf("Fsync %s", o.name)

	o.lock.Lock()
	defer o.lock.Unlock()

	r := fuse.ToStatus(syscall.Fsync(int(o.file.Fd())))

	// upload
	o.driver.Upload(o.name, o.file)

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
	log.Debugf("Chmod %s", o.name)
	// f.lock.Lock()
	// r := fuse.ToStatus(f.File.Chmod(os.FileMode(mode)))
	// f.lock.Unlock()

	// return r
	return fuse.OK
}

func (o *ObjectFile) Chown(uid uint32, gid uint32) fuse.Status {
	log.Debugf("Chown %s", o.name)
	// f.lock.Lock()
	// r := fuse.ToStatus(f.File.Chown(int(uid), int(gid)))
	// f.lock.Unlock()

	// return r
	return fuse.OK
}

func (o *ObjectFile) GetAttr(a *fuse.Attr) fuse.Status {

	st := syscall.Stat_t{}
	o.lock.Lock()
	err := syscall.Fstat(int(o.file.Fd()), &st)
	o.lock.Unlock()
	if err != nil {
		return fuse.ToStatus(err)
	}
	a.FromStat(&st)

	return fuse.OK

	return fuse.OK
}

const _UTIME_NOW = ((1 << 30) - 1)
const _UTIME_OMIT = ((1 << 30) - 2)

func (o *ObjectFile) Utimens(a *time.Time, m *time.Time) fuse.Status {
	log.Debugf("Utimes %s", o.name)

	var ts [2]syscall.Timespec

	if a == nil {
		ts[0].Nsec = _UTIME_OMIT
	} else {
		ts[0].Sec = a.Unix()
	}

	if m == nil {
		ts[1].Nsec = _UTIME_OMIT
	} else {
		ts[1].Sec = m.Unix()
	}

	o.lock.Lock()
	err := futimens(int(o.file.Fd()), &ts)
	o.lock.Unlock()
	return fuse.ToStatus(err)
}

// futimens - futimens(3) calls utimensat(2) with "pathname" set to null and
// "flags" set to zero
func futimens(fd int, times *[2]syscall.Timespec) (err error) {
	_, _, e1 := syscall.Syscall6(syscall.SYS_UTIMENSAT, uintptr(fd), 0, uintptr(unsafe.Pointer(times)), uintptr(0), 0, 0)
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	return
}
