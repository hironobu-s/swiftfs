package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/objfs"
)

func main() {
	app := newCliApp()
	app.RunAndExitOnError()
}

func newCliApp() *cli.App {
	app := cli.NewApp()
	app.Name = "swiftfsd"
	app.Version = "0.1alpha"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "mountpoint, m",
			Value: "",
			Usage: "The mount point for your file system.",
		},

		cli.StringFlag{
			Name:  "container-name, n",
			Value: "",
			Usage: "The container name.",
		},
	}

	app.Before = func(c *cli.Context) error {
		mountpoint := c.String("mountpoint")
		container := c.String("container-name")
		if mountpoint == "" || container == "" {
			return fmt.Errorf("Both arguments \"mountpoint\" and \"container-name\" are required.")
		}
		return nil
	}

	app.Action = action

	return app
}

func action(c *cli.Context) {

	log.SetLevel(log.DebugLevel)

	config := &objfs.Config{
		MountPoint:    c.String("mountpoint"),
		ContainerName: c.String("container-name"),
	}

	fs := objfs.NewObjFs(config)
	if err := fs.Mount(); err != nil {
		log.Debugf("Mount error: %v", err)
	}
}
