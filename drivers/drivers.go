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
	// GenerateFlags() []cli.Flag
	// ValidateFlags(*cli.Context) error

	Initialize() error
	List() []*Object
	Upload(string, io.ReadSeeker) error
	Delete(string) error
	Get(string) (*Object, error)
}
