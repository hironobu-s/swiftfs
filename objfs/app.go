package objfs

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func NewApp() *cli.App {
	app := cli.NewApp()

	app.Name = "objfs"
	app.Usage = "The file system to mount OpenStack Swift object storage via FUSE."
	app.Version = "0.1alpha"
	app.HideHelp = true
	app.Author = "Hironobu Saitoh"
	app.Email = "hiro@hironobu.org"

	config := NewConfig()
	app.Flags = config.GetFlags()

	app.Before = func(c *cli.Context) (err error) {
		return nil
	}

	app.Action = func(c *cli.Context) {
		if c.Bool("help") || len(c.Args()) == 0 {
			cli.ShowAppHelp(c)
			return
		}

		var err error
		if err = config.SetConfigFromContext(c); err != nil {
			log.Warnf("%v", err)
			return
		}

		fs := NewFileSystem(config.Driver, config.MountPoint)
		if err = fs.Mount(); err != nil {
			log.Warnf("%v", err)
		}
	}

	return app
}
