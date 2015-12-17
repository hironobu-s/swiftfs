package main

import (
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/hironobu-s/swiftfs/openstack"
)

func TestMain(m *testing.M) {
	c := config.NewConfig()
	c.CreateContainer = true
	c.Debug = true
	c.NoDaemon = true

	swift := openstack.NewSwift(c)
	if err := swift.Auth(); err != nil {
		log.Errorf("%v", err)
	}
}
