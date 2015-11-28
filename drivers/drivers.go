package drivers

import (
	"io"
	"time"

	"github.com/codegangsta/cli"
)

const (
	OBJECT = iota
	DIRECTORY
)

type Container struct {
	Name  string
	Quota uint64
	Used  uint64
	Count uint64
}

type ObjectList []Object

func (list ObjectList) Find(name string) *Object {
	for _, obj := range list {
		if obj.Name == name {
			return &obj
		}
	}
	return nil
}

type Object struct {
	Name         string
	Body         io.ReadCloser
	Size         uint64
	LastModified time.Time
	Type         int
}

type DriverConfig interface {
	GetFlags() []cli.Flag
	SetConfigFromContext(*cli.Context) error
}

type Driver interface {
	DriverName() string
	SetConfig(DriverConfig) error

	// Object handling
	Auth() error
	List() ObjectList
	Get(string) (Object, error)
	Upload(string, io.ReadSeeker) error
	Delete(string) error
	Copy(string, string) error

	// Directry handling
	MakeDirectory(string) error
	RemoveDirectory(string) error

	// Container handling
	GetContainer() (*Container, error)
	CreateContainer() error
	DeleteContainer() error
}
