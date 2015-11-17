package objfs

import (
	"fmt"
	"net/http"

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

	Driver        drivers.Driver
	drivers       map[string]drivers.Driver
	driverConfigs map[string]drivers.DriverConfig
}

func NewConfig() *Config {
	config := &Config{
		ObjectListSize: 1000,
	}

	config.loadDrivers()
	return config
}

func (c *Config) loadDrivers() {
	c.drivers = map[string]drivers.Driver{}
	c.driverConfigs = map[string]drivers.DriverConfig{}

	// TODO: Need driver auto detection.
	names := []string{"openstack"}

	for _, name := range names {
		switch name {
		case "openstack":
			c.drivers[name] = &openstack.Swift{}
			c.driverConfigs[name] = &openstack.SwiftConfig{}

			log.Infof("Load driver: %s", name)

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
			Usage: "Append debug logs to logfile instead of stdout.",
		},

		cli.StringFlag{
			Name:  "driver, d",
			Value: "openstack",
			Usage: "Set driver name of Object Storage.",
		},
	}
	flags = append(flags, fs...)

	// Merge driver-specific options
	for _, config := range c.driverConfigs {
		flags = append(flags, config.GetFlags()...)
	}

	return flags
}

func (c *Config) SetConfigFromContext(ctx *cli.Context) (err error) {
	c.Debug = ctx.Bool("debug")
	c.Logfile = ctx.String("logfile")
	driverName := ctx.String("driver")

	// Container name
	c.ContainerName = ctx.Args()[0]
	log.Infof("Container name: %s", c.ContainerName)

	// Mountpoint
	c.MountPoint = ctx.Args()[1]

	// Abs path of mountpoint
	if c.MountPoint, err = filepath.Abs(c.MountPoint); err != nil {
		return err
	}
	log.Infof("Mount point: %s", c.MountPoint)

	// Debug mode
	if c.Debug {
		log.Infof("Enable debug mode")

		log.SetLevel(log.DebugLevel)

		// Set LogTransport
		http.DefaultTransport = &drivers.DebugTransport{
			Transport: http.DefaultTransport,
		}
	}

	//  Detect drivers
	var ok bool
	c.Driver, ok = c.drivers[driverName]
	if !ok {
		return fmt.Errorf("Driver \"%s\" not found.", driverName)
	}
	log.Infof("%s driver detected", driverName)

	// Set driver config
	config, ok := c.driverConfigs[driverName]
	if !ok {
		return fmt.Errorf("DriverConfig \"%s\" not found.", driverName)
	}
	if err = config.SetConfigFromContext(ctx); err != nil {
		return err
	}
	c.Driver.SetConfig(config)

	log.Infof("Initialize driver config")

	return nil
}
