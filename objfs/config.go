package objfs

import (
	"fmt"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/drivers/openstack"
)

type Config struct {
	NoDaemon       bool
	MountPoint     string
	ContainerName  string
	DriverName     string
	ObjectListSize int
}

func NewConfig() *Config {
	config := &Config{
		MountPoint:     "tmp",
		ContainerName:  "test-container",
		ObjectListSize: 1000,
	}
	return config
}

func (c *Config) GetFlags() []cli.Flag {
	flags := make([]cli.Flag, 0, 100)

	fs := []cli.Flag{
		cli.StringFlag{
			Name:  "mountpoint, m",
			Value: "",
			Usage: "The mount point for your file system.",
		},

		cli.StringFlag{
			Name:  "driver, d",
			Value: "",
			Usage: "Driver name of object storage",
		},

		cli.StringFlag{
			Name:  "container-name, n",
			Value: "",
			Usage: "The container name.",
		},
	}

	flags = append(flags, fs...)

	// Drivers
	flags = append(flags, openstack.GenerateFlags()...)

	return flags
}

func (c *Config) SetConfigFromContext(ctx *cli.Context) (err error) {

	c.MountPoint = ctx.String("mountpoint")
	c.ContainerName = ctx.String("container-name")
	c.DriverName = ctx.String("driver")

	// Require
	var requires = make([]string, 0, 3)
	if c.MountPoint == "" {
		requires = append(requires, "mountpoint")
	}

	if c.ContainerName == "" {
		requires = append(requires, "container-name")
	}

	if c.DriverName == "" {
		requires = append(requires, "driver")
	}

	if len(requires) > 0 {
		return fmt.Errorf("Some of required parameters are provided. [%s]", strings.Join(requires, ","))
	}

	return nil
}
