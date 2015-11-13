package drivers

import (
	"io"
	"time"

	"github.com/codegangsta/cli"
)

type Container struct {
	Name string
}

type Object struct {
	Name         string
	Body         io.ReadCloser
	Size         uint64
	LastModified time.Time
}

type DriverConfig interface {
}

type Driver interface {
	GetFlags() []cli.Flag
	SetConfigFromContext(*cli.Context) error
	Initialize() error

	List() []*Object
	Upload(string, io.ReadSeeker) error
	Delete(string) error
	Get(string) (*Object, error)
}
