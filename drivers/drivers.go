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
	GetFlags() []cli.Flag
	SetConfigFromContext(*cli.Context) error
}

type Driver interface {
	DriverName() string
	SetConfig(DriverConfig) error

	Auth() error
	List() []Object
	Get(string) (Object, error)
	Upload(string, io.ReadSeeker) error
	Delete(string) error

	HasContainer() (bool, error)
	CreateContainer() error
	DeleteContainer() error
}
