package openstack

import (
	"io"
	"os"

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

func (s *Swift) GetFlags() []cli.Flag {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "os-user-id",
			Value: "",
			Usage: "[OpenStack] User ID",
		},
		cli.StringFlag{
			Name:  "os-username",
			Value: "",
			Usage: "[OpenStack] Username",
		},
		cli.StringFlag{
			Name:  "os-password",
			Value: "",
			Usage: "[OpenStack] Password",
		},
		cli.StringFlag{
			Name:  "os-tenant-id",
			Value: "",
			Usage: "[OpenStack] Tenant Id",
		},
		cli.StringFlag{
			Name:  "os-tenant-name",
			Value: "",
			Usage: "[OpenStack] Tenant Name",
		},
		cli.StringFlag{
			Name:  "os-auth-url",
			Value: "",
			Usage: "[OpenStack] Auth URL",
		},
	}
	return flags
}

func (s *Swift) SetConfigFromContext(ctx *cli.Context) (err error) {

	if os.Getenv("OS_AUTH_URL") != "" {
		log.Debugf("[OpenStack] Use auth parameters in ENV.")

		s.authOptions, err = openstack.AuthOptionsFromEnv()
		if err != nil {
			return err
		}

	} else {
		log.Debugf("[OpenStack] Use auth parameters via command-line options.")

		s.authOptions = gophercloud.AuthOptions{
			IdentityEndpoint: ctx.String("os-auth-url"),
			UserID:           ctx.String("os-user-id"),
			Username:         ctx.String("os-username"),
			Password:         ctx.String("os-password"),
			TenantID:         ctx.String("os-tenant-id"),
			TenantName:       ctx.String("os-tenant-name"),
		}
	}

	// Enable auto reauth
	s.authOptions.AllowReauth = true

	// container name
	s.containerName = ctx.String("container-name")

	return nil
}

func NewSwift() *Swift {
	cli := &Swift{}
	return cli
}

type Swift struct {
	containerName  string
	objectListSize int
	authOptions    gophercloud.AuthOptions

	client *gophercloud.ServiceClient

	objects []drivers.Object
}

func (s *Swift) Initialize() error {

	if s.authOptions.Username != "" {
		log.Debugf("[OpenStack] Authenticating by username(%s)", s.authOptions.Username)
	} else {
		log.Debugf("[OpenStack] Authenticating by user-id(%s)", s.authOptions.UserID)
	}

	provider, err := openstack.AuthenticatedClient(s.authOptions)
	if err != nil {
		return err
	}

	s.client, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{})
	if err != nil {
		return err
	}

	return nil
}

func (s *Swift) List() (objects []*drivers.Object) {

	pager := swiftobjects.List(s.client, s.containerName, swiftobjects.ListOpts{
		Full: true,
	})

	objects = make([]*drivers.Object, s.objectListSize)
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

	log.Debugf("[OpenStack] Fetch object list. number of objects is %d.", i)
	return objects
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

func (s *Swift) Get(name string) (obj *drivers.Object, err error) {

	opts := swiftobjects.DownloadOpts{}

	log.Debugf("[OpenStack] Download object named \"%s\"", name)

	result := swiftobjects.Download(s.client, s.containerName, name, opts)
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

func (s *Swift) CreateContainer(name string) error {
	opts := swiftcontainers.CreateOpts{}
	result := swiftcontainers.Create(s.client, name, opts)
	return result.Err
}

func (s *Swift) DeleteContainer(name string) error {
	result := swiftcontainers.Delete(s.client, name)
	return result.Err
}

func (s *Swift) ListContainer() {

	pager := swiftcontainers.List(s.client, swiftcontainers.ListOpts{Full: true})
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
