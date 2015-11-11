package objfs

import (
	"github.com/codegangsta/cli"
)

type Config struct {
	MountPoint     string
	ContainerName  string
	ObjectListSize int
}

func NewConfig(flags []cli.Flag) *Config {

	config := &Config{
		MountPoint:     "tmp",
		ContainerName:  "test-container",
		ObjectListSize: 1000,
	}
	return config
}
