package objfs

import (
	"fmt"
	"net/http"
	"os"

	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/drivers"
	"github.com/hironobu-s/objfs/drivers/openstack"
)

type Config struct {
	Debug           bool
	NoDaemon        bool
	Logfile         *os.File // Need close() after use
	MountPoint      string
	ContainerName   string
	CreateContainer bool
	ObjectListSize  int

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

	// TODO: driver auto detection.
	names := []string{"openstack"}

	for _, name := range names {
		switch name {
		case "openstack":
			c.drivers[name] = &openstack.Swift{}
			c.driverConfigs[name] = &openstack.SwiftConfig{}

			log.Debugf("Load driver: %s", name)

		default:
			log.Warnf("Driver \"%s\" not found.", name)
		}
	}
}

func (c *Config) GetFlags() []cli.Flag {
	flags := make([]cli.Flag, 0, 100)

	// Global options
	fs := []cli.Flag{
		cli.HelpFlag,

		cli.BoolFlag{
			Name:  "debug",
			Usage: "Output debug information",
		},

		cli.BoolFlag{
			Name:  "no-daemon",
			Usage: "Start an objfs process as a foreground (for debugging)",
		},

		cli.StringFlag{
			Name:  "logfile, l",
			Usage: "The logfile name that appends some information instead of stdout/stderr",
		},

		cli.StringFlag{
			Name:  "driver, d",
			Value: "openstack",
			Usage: "The driver name of the Object Storage (currently, the only supported driver is \"openstack\")",
		},

		cli.BoolFlag{
			Name:  "create-container, c",
			Usage: "Create a container if is not exist",
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

	// Debug mode
	c.Debug = ctx.Bool("debug")
	if c.Debug {
		log.SetLevel(log.DebugLevel)

		// Set LogTransport
		http.DefaultTransport = &drivers.DebugTransport{
			Transport: http.DefaultTransport,
		}

	} else {
		log.SetLevel(log.WarnLevel)
	}

	// logfile
	var logfile = ctx.String("logfile")
	if logfile != "" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		c.Logfile = f

		log.SetFormatter(&LogfileFormatter{})
		log.SetOutput(f)

		log.Debugf(" to %s", logfile)

	} else {
		c.Logfile = nil
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:    true,
			DisableTimestamp: false,
			TimestampFormat:  "Jan 02 15:04:05",
		})
	}

	// Create Container
	c.CreateContainer = ctx.Bool("create-container")

	// Container name
	c.ContainerName = ctx.Args()[0]
	log.Debugf("Container name: %s", c.ContainerName)

	// Mountpoint
	c.MountPoint = ctx.Args()[1]
	if c.MountPoint, err = filepath.Abs(c.MountPoint); err != nil {
		return err
	}
	log.Debugf("Mount point: %s", c.MountPoint)

	// No daemon mode
	c.NoDaemon = ctx.Bool("no-daemon")
	if c.NoDaemon {
		log.Debug("no-daemon option is enabled")
	}

	//  Detect drivers
	driverName := ctx.String("driver")

	var ok bool
	c.Driver, ok = c.drivers[driverName]
	if !ok {
		return fmt.Errorf("Driver \"%s\" not found.", driverName)
	}
	log.Debugf("%s driver detected", driverName)

	// Set driver config
	config, ok := c.driverConfigs[driverName]
	if !ok {
		return fmt.Errorf("DriverConfig \"%s\" not found.", driverName)
	}
	if err = config.SetConfigFromContext(ctx); err != nil {
		return err
	}
	c.Driver.SetConfig(config)

	log.Debugf("Initialize driver config")

	return nil
}
