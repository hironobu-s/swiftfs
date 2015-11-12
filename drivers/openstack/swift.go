package openstack

import (
	"io"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/hironobu-s/objfs/drivers"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	swiftcontainers "github.com/rackspace/gophercloud/openstack/objectstorage/v1/containers"
	swiftobjects "github.com/rackspace/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/rackspace/gophercloud/pagination"

	_ "github.com/motemen/go-loghttp/global"
)

type SwiftConfig struct {
	ContainerName  string
	ObjectListSize int
}

func (c *SwiftConfig) GetFlags() []cli.Flag {
	flags := make([]cli.Flag, 0)
	return flags
}

type SwiftClient struct {
	config *SwiftConfig

	client *gophercloud.ServiceClient

	objects []drivers.Object
}

func NewSwiftClient(config *SwiftConfig) *SwiftClient {
	cli := &SwiftClient{
		config: config,
	}
	return cli
}

func (c *SwiftClient) Initialize() error {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return err
	}

	// Enable reauth
	opts.AllowReauth = true

	log.Debugf("[SWIFT] Authenticating for OpenStack user(%s)...", opts.Username)
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return err
	}

	c.client, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}

	return nil
}

func (c *SwiftClient) List() (objects []*drivers.Object) {

	pager := swiftobjects.List(c.client, c.config.ContainerName, swiftobjects.ListOpts{
		Full: true,
	})

	objects = make([]*drivers.Object, c.config.ObjectListSize)
	var i = 0
	pager.EachPage(func(page pagination.Page) (bool, error) {
		objlist, err := swiftobjects.ExtractInfo(page)
		if err != nil {
			log.Debugf("%v\n", err)
			return false, err
		}

		for _, o := range objlist {
			log.Debugln("append list " + o.Name)

			// gophercloudがタイムゾーンを考慮しないで返してくるっぽい？
			var lastmodified time.Time

			o.LastModified += "Z"
			lastmodified, err := time.Parse(time.RFC3339, o.LastModified)
			if err != nil {
				log.Debugf("Invalid time format[%s]", o.LastModified)
				lastmodified = time.Now()
			}

			obj := &drivers.Object{
				Name:         o.Name,
				Body:         nil,
				Size:         uint64(o.Bytes),
				LastModified: lastmodified,
			}

			objects = append(objects, obj)
			i++
		}

		return true, nil
	})

	log.Debugf("[SWIFT] Fetch object list. number of objects is %d.", i)
	return objects
}

func (c *SwiftClient) Upload(name string, data io.ReadSeeker) error {
	opts := swiftobjects.CreateOpts{}
	result := swiftobjects.Create(c.client, c.config.ContainerName, name, data, opts)
	if result.Err != nil {
		return result.Err
	}
	return nil
}

func (c *SwiftClient) Delete(name string) error {
	result := swiftobjects.Delete(c.client, c.config.ContainerName, name, nil)
	return result.Err
}

func (c *SwiftClient) Get(name string) (obj *drivers.Object, err error) {

	opts := swiftobjects.DownloadOpts{}

	log.Debugf("[SWIFT] Download object named \"%s\"", name)

	result := swiftobjects.Download(c.client, c.config.ContainerName, name, opts)
	if result.Err != nil {
		return nil, err
	}

	headers, err := result.Extract()
	if err != nil {
		return nil, err
	}

	obj = &drivers.Object{
		Name:         name,
		Body:         result.Body,
		Size:         uint64(headers.ContentLength),
		LastModified: headers.LastModified,
	}

	return obj, nil
}

func (c *SwiftClient) CreateContainer(name string) error {
	opts := swiftcontainers.CreateOpts{}
	result := swiftcontainers.Create(c.client, name, opts)
	return result.Err
}

func (c *SwiftClient) DeleteContainer(name string) error {
	result := swiftcontainers.Delete(c.client, name)
	return result.Err
}

func (c *SwiftClient) ListContainer() {

	pager := swiftcontainers.List(c.client, swiftcontainers.ListOpts{Full: true})
	pager.EachPage(func(page pagination.Page) (bool, error) {
		containerList, err := swiftcontainers.ExtractInfo(page)
		if err != nil {
			return false, err
		}

		for _, c := range containerList {
			println(c.Name)
		}
		return true, nil
	})
}
