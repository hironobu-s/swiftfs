package objfs

import (
	"fmt"

	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/drivers"
	"github.com/hironobu-s/objfs/drivers/openstack"
)

type Config struct {
	Debug          bool
	NoDaemon       bool
	Logfile        string
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
			Usage: "Output more informations",
		},

		cli.BoolFlag{
			Name:  "daemon",
			Usage: "Run as a daemon mode. (default=true)",
		},

		cli.StringFlag{
			Name:  "logfile, l",
			Usage: "Logfile name",
		},

		cli.StringFlag{
			Name:  "driver, d",
			Value: "openstack",
			Usage: "Driver name of object storage",
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

	c.Logfile = ctx.String("logfile")
	driverName := ctx.String("driver")

	// Container name
	c.ContainerName = ctx.Args()[0]

	// Mountpoint
	c.MountPoint = ctx.Args()[1]

	// Abs path of mountpoint
	if c.MountPoint, err = filepath.Abs(c.MountPoint); err != nil {
		return err
	}

	// debug mode
	if c.Debug {
		log.SetLevel(log.DebugLevel)
	}

	//  load and detect drivers
	c.loadDrivers()

	d, ok := c.drivers[driverName]
	if !ok {
		return fmt.Errorf("Driver \"%s\" not found.", driverName)
	}
	c.Driver = d

	c.Driver.SetConfigFromContext(ctx)

	return nil
}
