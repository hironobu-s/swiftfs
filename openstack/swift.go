package openstack

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hironobu-s/swiftfs/config"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/objectstorage/v1/accounts"
	swiftcontainers "github.com/rackspace/gophercloud/openstack/objectstorage/v1/containers"
	swiftobjects "github.com/rackspace/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/rackspace/gophercloud/pagination"
)

const (
	DEFAULT_ACCOUNT_QUOTA = 1024 * 1024 * 1024 * 1024 * 100 // 100TB
)

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

func (s *Swift) List() (list *ObjectList) {
	pager := swiftobjects.List(s.client, s.containerName, swiftobjects.ListOpts{
		Full: true,
	})

	list = NewObjectList()
	var i = 0
	pager.EachPage(func(page pagination.Page) (bool, error) {
		objlist, err := swiftobjects.ExtractInfo(page)
		if err != nil {
			log.Debugf("%v\n", err)
			return false, err
		}

		for _, o := range objlist {
			//log.Debugf("append object %s", o.Name)

			// gophercloudがタイムゾーンを考慮しないで返してくるっぽい？
			var lastmodified time.Time

			o.LastModified += "Z"
			lastmodified, err := time.Parse(time.RFC3339, o.LastModified)
			if err != nil {
				log.Debugf("Invalid time format[%s]", o.LastModified)
				lastmodified = time.Now()
			}

			var t int
			if o.ContentType == "application/directory" {
				t = DIRECTORY
			} else {
				t = FILE
			}

			list.Set(o.Name, uint64(o.Bytes), lastmodified, t)
			i++
		}

		return true, nil
	})

	log.Debugf("(OpenStack) Fetch object list. number of objects is %d.", i)
	return list
}

func (s *Swift) Upload(name string, data io.ReadSeeker) error {
	opts := swiftobjects.CreateOpts{}
	result := swiftobjects.Create(s.client, s.containerName, name, data, opts)
	if result.Err != nil {
		return result.Err
	}
	return nil
}

func (s *Swift) Delete(name string) error {
	result := swiftobjects.Delete(s.client, s.containerName, name, nil)
	return result.Err
}

func (s *Swift) Get(name string) (obj Object, err error) {
	obj = Object{}

	log.Debugf("(OpenStack) Download object (%s)", name)
	opts := swiftobjects.DownloadOpts{}
	result := swiftobjects.Download(s.client, s.containerName, name, opts)
	if result.Err != nil {
		return obj, err
	}

	headers, err := result.Extract()
	if err != nil {
		return obj, err
	}

	obj.Name = name
	obj.Body = result.Body
	obj.Size = uint64(headers.ContentLength)
	obj.LastModified = headers.LastModified

	return obj, nil
}

func (s *Swift) GetContainer() (container *Container, err error) {
	m := new(sync.Mutex)
	cerr := make(chan error)
	container = &Container{
		Name: s.containerName,
	}

	// get account quota
	go func(mm *sync.Mutex) {
		account := accounts.Get(s.client, nil)
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

		result := swiftcontainers.Get(s.client, s.containerName)
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
			return nil, err
		}
		i++
	}

	return container, nil
}

func (s *Swift) Copy(oldName string, newName string) error {
	log.Debugf("(OpenStack) Copy object from \"%s\" to \"%s\"", oldName, newName)

	opts := swiftobjects.CopyOpts{
		Destination: fmt.Sprintf("%s/%s", s.containerName, newName),
	}
	result := swiftobjects.Copy(s.client, s.containerName, oldName, opts)
	return result.Err
}

func (s *Swift) CreateContainer() error {
	opts := swiftcontainers.CreateOpts{}
	result := swiftcontainers.Create(s.client, s.containerName, opts)
	return result.Err
}

func (s *Swift) DeleteContainer() error {
	result := swiftcontainers.Delete(s.client, s.containerName)
	return result.Err
}

func (s *Swift) MakeDirectory(name string) error {
	opts := swiftobjects.CreateOpts{
		ContentType: "application/directory",
	}
	result := swiftobjects.Create(s.client, s.containerName, name, strings.NewReader(""), opts)
	return result.Err
}

func (s *Swift) RemoveDirectory(name string) error {
	opts := swiftobjects.DeleteOpts{}
	result := swiftobjects.Delete(s.client, s.containerName, name, opts)
	return result.Err
}
