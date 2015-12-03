package openstack

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/rackspace/gophercloud/pagination"
	"github.com/rackspace/gophercloud/rackspace/objectstorage/v1/accounts"
	"github.com/rackspace/gophercloud/rackspace/objectstorage/v1/containers"
)

const (
	DEFAULT_ACCOUNT_QUOTA = 1024 * 1024 * 1024 * 1024 * 100 // 100TB
)

type SwiftObject struct {
	objects.Object
}

type Swift struct {
	client *gophercloud.ServiceClient

	containerName   string
	ObjectListSize  int
	authOptions     gophercloud.AuthOptions
	endpointOptions gophercloud.EndpointOpts
}

func NewSwift(c *config.Config) *Swift {
	s := &Swift{}

	// Auth options
	var err error
	s.authOptions, err = openstack.AuthOptionsFromEnv()
	if err == nil {
		log.Debugf("(OpenStack) Use auth parameters in ENV.")

	} else {
		log.Debugf("(OpenStack) Use auth parameters via command-line options.")
		s.authOptions = gophercloud.AuthOptions{
			IdentityEndpoint: c.IdentityEndpoint,
			UserID:           c.UserID,
			Username:         c.Username,
			Password:         c.Password,
			TenantID:         c.TenantID,
			TenantName:       c.TenantName,
		}
	}

	// Endpoint options
	s.endpointOptions = gophercloud.EndpointOpts{}
	if c.RegionName != "" {
		s.endpointOptions.Region = c.RegionName
	}

	// Enable auto reauth
	s.authOptions.AllowReauth = true

	// Container Name
	s.containerName = c.ContainerName

	return s
}

func (s *Swift) Auth() error {
	if s.authOptions.Username != "" {
		log.Debugf("(OpenStack) Authenticate by username(%s)", s.authOptions.Username)
	} else if s.authOptions.UserID != "" {
		log.Debugf("(OpenStack) Authenticate by user-id(%s)", s.authOptions.UserID)
	} else {
		log.Debugf("(OpenStack) Authenticate")
	}

	provider, err := openstack.AuthenticatedClient(s.authOptions)
	if err != nil {
		return err
	}

	s.client, err = openstack.NewObjectStorageV1(provider, s.endpointOptions)
	if err != nil {
		return err
	}

	return nil
}

func (s *Swift) List() (objch chan objects.Object, n chan int) {
	objch = make(chan objects.Object)
	n = make(chan int)

	go func() {
		pager := objects.List(s.client, s.containerName, objects.ListOpts{
			Full: true,
		})

		i := 0
		pager.EachPage(func(page pagination.Page) (bool, error) {
			objlist, err := objects.ExtractInfo(page)
			if err != nil {
				log.Debugf("%v\n", err)
				return false, err
			}

			for _, obj := range objlist {
				objch <- obj
				i++
			}

			return true, nil
		})

		n <- i
	}()

	return objch, n
}

func (s *Swift) Upload(name string, data io.ReadSeeker) error {
	opts := objects.CreateOpts{}
	result := objects.Create(s.client, s.containerName, name, data, opts)
	if result.Err != nil {
		return result.Err
	}
	return nil
}

func (s *Swift) Delete(name string) error {
	result := objects.Delete(s.client, s.containerName, name, nil)
	return result.Err
}

func (s *Swift) Get(name string) objects.DownloadResult {
	log.Debugf("(OpenStack) Download object (%s)", name)
	opts := objects.DownloadOpts{}
	return objects.Download(s.client, s.containerName, name, opts)
}

func (s *Swift) Copy(oldName string, newName string) error {
	log.Debugf("(OpenStack) Copy object from \"%s\" to \"%s\"", oldName, newName)

	opts := objects.CopyOpts{
		Destination: fmt.Sprintf("%s/%s", s.containerName, newName),
	}
	result := objects.Copy(s.client, s.containerName, oldName, opts)
	return result.Err
}

type Container struct {
	Quota uint64
	Used  uint64
	Count uint64
}

func (s *Swift) GetContainer() (container Container, err error) {
	cerr := make(chan error)
	m := &sync.Mutex{}
	container = Container{}

	// get account quota
	go func(mm *sync.Mutex) {
		account := accounts.Get(s.client)
		headers, err := account.ExtractHeader()
		if err != nil {
			cerr <- err
			return
		}

		var quota uint64
		strval := headers.Get("X-Account-Meta-Quota-Bytes")
		if strval != "" {
			quota, err = strconv.ParseUint(strval, 10, 64)
			if err != nil {
				quota = DEFAULT_ACCOUNT_QUOTA
			}

		} else {
			quota = DEFAULT_ACCOUNT_QUOTA
		}

		m.Lock()
		container.Quota = quota
		m.Unlock()

		cerr <- nil
	}(m)

	// get container used
	go func(mm *sync.Mutex) {
		var strval string

		result := containers.Get(s.client, s.containerName)
		if result.Err != nil {
			cerr <- result.Err
			return
		}

		headers, err := result.ExtractHeader()
		if err != nil {
			cerr <- err
			return
		}

		var used, count uint64
		strval = headers.Get("X-Container-Bytes-Used")
		if strval != "" {
			if used, err = strconv.ParseUint(strval, 10, 64); err != nil {
				used = 0
			}
		}

		strval = headers.Get("X-Container-Object-Count")
		if strval != "" {
			if count, err = strconv.ParseUint(strval, 10, 64); err != nil {
				count = 0
			}
		}

		m.Lock()
		container.Used = used
		container.Count = count
		m.Unlock()

		cerr <- nil
	}(m)

	var i = 0
	for i < 2 {
		if err = <-cerr; err != nil {
			return container, err
		}
		i++
	}

	return container, nil
}

func (s *Swift) CreateContainer() error {
	opts := containers.CreateOpts{}
	result := containers.Create(s.client, s.containerName, opts)
	return result.Err
}

func (s *Swift) DeleteContainer() error {
	result := containers.Delete(s.client, s.containerName)
	return result.Err
}

func (s *Swift) MakeDirectory(name string) error {
	opts := objects.CreateOpts{
		ContentType: "application/directory",
	}
	result := objects.Create(s.client, s.containerName, name, strings.NewReader(""), opts)
	return result.Err
}

func (s *Swift) RemoveDirectory(name string) error {
	opts := objects.DeleteOpts{}
	result := objects.Delete(s.client, s.containerName, name, opts)
	return result.Err
}
