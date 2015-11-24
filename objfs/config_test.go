package objfs

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codegangsta/cli"
)

func TestMain(m *testing.M) {
	code := m.Run()
	defer os.Exit(code)
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	if config.ObjectListSize != 1000 {
		t.Errorf("Invalid value set to ObjectListSize")
	}
}

func TestGetFlags(t *testing.T) {
	config := NewConfig()
	flags := config.GetFlags()

	hasOpenStack := false
	for _, f := range flags {
		if strings.HasPrefix(f.String(), "--os-") {
			hasOpenStack = true
		}
	}

	if !hasOpenStack {
		t.Errorf("OpenStack driver does not loaded")
	}
}

func TestSetConfigFromContext(t *testing.T) {
	config := NewConfig()

	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, f := range config.GetFlags() {
		f.Apply(set)
	}

	testargs := []string{
		"--debug",
		"--no-daemon",
		"--logfile=log.txt",
		"--driver=openstack",
		"--create-container",
		"testcontainer",
		"testmountpoint",
	}

	set.Parse(testargs)
	c := cli.NewContext(nil, set, nil)
	if err := config.SetConfigFromContext(c); err != nil {
		t.Errorf("%v", err)
	}

	if !config.Debug {
		t.Errorf("The config parameter \"Debug\" is false in spite of --debug flag is specified.")
	}

	if !config.NoDaemon {
		t.Errorf("The config parameter \"NoDaemon\" is false in spite of --no-daemon flag is specified.")
	}

	if !config.CreateContainer {
		t.Errorf("The config parameter \"CreateContainer\" is false in spite of --create-container flag is specified.")
	}

	if config.Logfile == nil {
		t.Errorf("The config parameter \"Logfile\" is null in spite of --logfile flag is specified")
	}

	if config.ContainerName != "testcontainer" {
		t.Errorf("The config parameter \"ContainerName\" is incorrect [%s]", config.ContainerName)
	}

	if filepath.Base(config.MountPoint) != "testmountpoint" {
		t.Errorf("The config parameter \"MountPoint\" is incorrect [%s]", filepath.Base(config.MountPoint))
	}

	if config.Driver.DriverName() != "OpenStack Swift" {
		t.Errorf("DriverName is incorrect [%s]", config.Driver.DriverName())
	}
}
