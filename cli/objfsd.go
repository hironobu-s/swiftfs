package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/objfs"
)

func main() {
	app := newCliApp()
	app.RunAndExitOnError()
}

var config *objfs.Config

func newCliApp() *cli.App {
	config = &objfs.Config{}

	app := cli.NewApp()
	app.Name = "swiftfsd"
	app.Version = "0.1alpha"
	app.HideHelp = true
	app.Author = "Hironobu Saitoh"
	app.Email = "hiro@hironobu.org"

	app.Flags = config.GetFlags()
	app.Before = config.SetConfigFromContext

	app.Action = action

	return app
}

func action(c *cli.Context) {

	log.SetLevel(log.DebugLevel)

	fs := objfs.NewObjFs(config)
	if err := fs.Mount(); err != nil {
		log.Debugf("Mount error: %v", err)
	}
}
