package objfs

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/drivers"
	"github.com/hironobu-s/objfs/drivers/openstack"
)

type Config struct {
	Debug          bool
	NoDaemon       bool
	MountPoint     string
	ContainerName  string
	ObjectListSize int

	Driver  drivers.Driver
	drivers map[string]drivers.Driver
}

func NewConfig() *Config {
	config := &Config{
		ObjectListSize: 1000,
	}

	return config
}

func (c *Config) loadDrivers() {
	c.drivers = map[string]drivers.Driver{}

	// TODO: Need driver auto detection.
	names := []string{"openstack"}

	for _, name := range names {
		switch name {
		case "openstack":
			c.drivers[name] = openstack.NewSwift()

		default:
			log.Warnf("Driver \"%s\" not found.", name)
			continue
		}

		log.Debugf("Driver \"%s\" loaded.", name)
	}
}

func (c *Config) GetFlags() []cli.Flag {
	flags := make([]cli.Flag, 0, 100)

	// Global options
	fs := []cli.Flag{
		cli.HelpFlag,

		cli.BoolFlag{
			Name:  "debug",
			Usage: "Debug mode.",
		},

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

	// Merge driver-specific options
	for _, d := range c.drivers {
		flags = append(flags, d.GetFlags()...)
	}

	return flags
}

func (c *Config) SetConfigFromContext(ctx *cli.Context) (err error) {

	c.Debug = ctx.Bool("debug")
	c.MountPoint = ctx.String("mountpoint")
	c.ContainerName = ctx.String("container-name")
	driverName := ctx.String("driver")

	// Validate required options
	var requires = make([]string, 0, 3)
	if c.MountPoint == "" {
		requires = append(requires, "mountpoint")
	}

	if c.ContainerName == "" {
		requires = append(requires, "container-name")
	}

	if driverName == "" {
		requires = append(requires, "driver")
	}

	if len(requires) > 0 {
		return fmt.Errorf("Some of required parameters are provided. [%s]", strings.Join(requires, ","))
	}

	if c.Debug {
		log.SetLevel(log.DebugLevel)
	}

	// Detect driver
	c.loadDrivers()

	d, ok := c.drivers[driverName]
	if !ok {
		return fmt.Errorf("Driver \"%s\" not found.", driverName)
	}
	c.Driver = d

	c.Driver.SetConfigFromContext(ctx)

	return nil
}
