package dummy

import (
	"io"

	"fmt"
	"time"

	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/drivers"
)

type DummyConfig struct {
	NumListObject int
}

func (DummyConfig) GetFlags() []cli.Flag {
	return []cli.Flag{}
}

func (DummyConfig) SetConfigFromContext(c *cli.Context) error {
	return nil
}

// -----

type Dummy struct {
	config *DummyConfig
}

func (d *Dummy) DriverName() string {
	return "Dummy"
}

func (d *Dummy) SetConfig(c drivers.DriverConfig) (err error) {
	var ok bool
	d.config, ok = c.(*DummyConfig)
	if !ok {
		return fmt.Errorf("Can't convert an argument to DummyConfig")
	}

	d.config.NumListObject = 100
	return nil
}

func (d *Dummy) Auth() error {
	return nil
}

func (d *Dummy) List() []*drivers.Object {
	i := 0
	list := make([]*drivers.Object, d.config.NumListObject)
	for i < d.config.NumListObject {
		name := fmt.Sprintf("dummy-object%05d", i)
		list[i] = &drivers.Object{
			Name:         name,
			Body:         nil,
			Size:         0,
			LastModified: time.Now(),
		}
		i++
	}
	return list
}

func (d *Dummy) Get(string) (*drivers.Object, error) {
	return &drivers.Object{
		Name:         "dummy-object",
		Body:         nil,
		Size:         0,
		LastModified: time.Now(),
	}, nil
}

func (d *Dummy) Upload(string, io.ReadSeeker) error {
	return nil
}

func (d *Dummy) Delete(string) error {
	return nil
}

func (d *Dummy) HasContainer() (bool, error) {
	return true, nil
}

func (d *Dummy) CreateContainer() error {
	return nil
}

func (d *Dummy) DeleteContainer() error {
	return nil
}
