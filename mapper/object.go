package mapper

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hironobu-s/swiftfs/openstack"
)

const (
	FILE = iota
	DIRECTORY
)

type Object interface {
	Localpath() string
	Open(flag int, perm os.FileMode) (*os.File, error)
	Flush() error
	Upload() error
	download() error
}

type object struct {
	Path string // foo/bar/buz.txt
	Name string // buz.txt
	Dir  string // foo/bat
	Type int    // const FILE or DIRECTORY

	Size  uint64
	Mtime time.Time

	swift      *openstack.Swift
	downloaded bool
}

func (o *object) Localpath() string {
	p := strings.Replace(o.Path, "/", "-", -1)
	return filepath.Join(os.TempDir(), "swiftfs", p)
}

// Open Temporary file
// Need to call close() after useing.
func (o *object) Open(flag int, perm os.FileMode) (*os.File, error) {

	// Download the filedata from the object storage when filesystem try to open an localfile
	// But, it does not need to download if O_TRUNC or O_CREATE flag passed.
	_, err := os.Stat(o.Localpath())
	if (flag&os.O_TRUNC == 0 && flag&os.O_CREATE == 0) && err != nil {
		log.Debugf("Open temporary file %s with downloading flag:%d %v", o.Path, flag, err)
		if err := o.download(); err != nil {
			log.Warnf("Download error %s, %v", o.Path, err)
			return nil, err
		}
		o.downloaded = true

	} else {
		log.Debugf("Open temporary file %s flag:%d", o.Path, flag)
	}

	file, err := os.OpenFile(o.Localpath(), flag, perm)
	if err != nil {
		return nil, err
	}

	return file, err
}

func (o *object) download() error {
	// Do not use o.Open() method.
	file, err := os.OpenFile(o.Localpath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	result := o.swift.Get(o.Path)
	defer result.Body.Close()

	if _, err = io.Copy(file, result.Body); err != nil {
		return err
	}

	return nil
}

func (o *object) Flush() (err error) {
	// Do not use o.Open() method.
	file, err := os.OpenFile(o.Localpath(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// update stat
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	o.Size = uint64(stat.Size())
	o.Mtime = stat.ModTime()

	return nil
}

func (o *object) Upload() (err error) {
	// Do not use o.Open() method.
	file, err := os.OpenFile(o.Localpath(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Flush
	o.Flush()

	// upload to object storage
	return o.swift.Upload(o.Path, file)
}

func newObject(swift *openstack.Swift, path string, t int) (obj *object) {
	name := filepath.Base(path)
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}

	obj = &object{
		Path: path,
		Name: name,
		Dir:  dir,
		Type: t,

		Size:  0,
		Mtime: time.Now(),

		swift:      swift,
		downloaded: false,
	}
	return obj
}
