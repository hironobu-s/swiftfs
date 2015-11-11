package drivers

import (
	"io"
	"time"

	"github.com/codegangsta/cli"
)

type DriverConfig interface {
	GetFlags() []cli.Flag
}

type Container struct {
	Name string
}

type Object struct {
	Name         string
	Body         io.ReadCloser
	Size         uint64
	LastModified time.Time
}

type Driver interface {
	//Containers() []Container
	Initialize() error
	List() []*Object
	Upload(name string, data io.ReadSeeker) error
	Delete(name string) error
	Get(name string) (*Object, error)
}
