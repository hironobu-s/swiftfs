package config

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

const (
	APP_VERSION = "0.1.0"
	APP_NAME    = "swiftfs"
)

type Config struct {
	Debug           bool
	NoDaemon        bool
	Logfile         *os.File // Need close() after use
	MountPoint      string
	CreateContainer bool

	// OpenStack credential
	IdentityEndpoint string
	UserID           string
	Username         string
	Password         string
	TenantID         string
	TenantName       string
	RegionName       string

	// Container
	ContainerName string

	// Size of internal slice that includes the objects which from Object Storage.
	// This parameter affect the performance to build it.
	ObjectListSize int
}

func NewConfig() *Config {
	config := &Config{
		ObjectListSize: 1000,
	}

	return config
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
			Usage: "Start an swiftfs process as a foreground (for debugging)",
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

		cli.StringFlag{
			Name:   "os-user-id",
			Value:  "",
			Usage:  "(OpenStack) User ID",
			EnvVar: "OS_USERID",
		},
		cli.StringFlag{
			Name:   "os-username",
			Value:  "",
			Usage:  "(OpenStack) Username",
			EnvVar: "OS_USERNAME",
		},
		cli.StringFlag{
			Name:   "os-password",
			Value:  "",
			Usage:  "(OpenStack) Password",
			EnvVar: "OS_PASSWORD",
		},
		cli.StringFlag{
			Name:   "os-tenant-id",
			Value:  "",
			Usage:  "(OpenStack) Tenant Id",
			EnvVar: "OS_TENANT_ID",
		},
		cli.StringFlag{
			Name:   "os-tenant-name",
			Value:  "",
			Usage:  "(OpenStack) Tenant Name",
			EnvVar: "OS_TENANT_NAME",
		},
		cli.StringFlag{
			Name:   "os-auth-url",
			Value:  "",
			Usage:  "(OpenStack) Auth URL(required)",
			EnvVar: "OS_AUTH_URL",
		},
		cli.StringFlag{
			Name:   "os-region-name",
			Value:  "",
			Usage:  "(OpenStack) Region Name",
			EnvVar: "OS_REGION_NAME",
		},
	}
	flags = append(flags, fs...)

	return flags
}

func (c *Config) SetConfigFromContext(ctx *cli.Context) (err error) {
	// Debug mode
	c.Debug = ctx.Bool("debug")
	if c.Debug {
		log.SetLevel(log.DebugLevel)

		// Set LogTransport
		http.DefaultTransport = &DebugTransport{
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

	// OpenStack
	c.IdentityEndpoint = ctx.String("os-auth-url")
	if c.IdentityEndpoint == "" {
		return fmt.Errorf("You must provide os-auth-url")
	}

	c.UserID = ctx.String("os-user-id")
	c.Username = ctx.String("os-username")
	c.Password = ctx.String("os-password")
	c.TenantID = ctx.String("os-tenant-id")
	c.TenantName = ctx.String("os-tenant-name")
	c.RegionName = ctx.String("os-region-name")

	c.ContainerName = ctx.Args()[0]
	if c.ContainerName == "" {
		return fmt.Errorf("Container name was not provided.")
	}

	// Default 1000
	c.ObjectListSize = 1000

	return nil
}
