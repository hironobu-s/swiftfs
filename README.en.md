# SwiftFS

SwiftFS is the file system to mount "Swift" OpenStack Object Storage via FUSE. This product targets for Unix flatforms.

This product may works on some OpenStack environments. We are testing on the following platforms.

- ConoHa Object Storage (https://www.conoha.jp/)
- Rackspace Cloud Files (http://www.rackspace.com/cloud/files)

## Install

Please download an executable file from [GitHub Release](https://github.com/hironobu-s/swiftfs/releases).

### Linux(amd64)

```shell
F=swiftfs curl -sL https://github.com/hironobu-s/swiftfs/releases/download/current/swiftfs.amd64.gz | zcat > $F && chmod +x $F
```


## Use

### Authentication 

You'll need to set some options for authentication with OpenStack APIs, 

**Via command-line arguments**

You can use options begining with "os-" to authenticate. 

```shell
$ swiftfs -h
--os-user-id                 (OpenStack) User ID [$OS_USERID]
--os-username                (OpenStack) Username [$OS_USERNAME]
--os-password                (OpenStack) Password [$OS_PASSWORD]
--os-tenant-id               (OpenStack) Tenant Id [$OS_TENANT_ID]
--os-tenant-name             (OpenStack) Tenant Name [$OS_TENANT_NAME]
--os-auth-url                (OpenStack) Auth URL(required) [$OS_AUTH_URL]
--os-region-name             (OpenStack) Region Name [$OS_REGION_NAME]
```

**Via environment variables**

Setting environment variables to authenticate.

```
export OS_USERID=***
export OS_USERNAME=***
export OS_PASSWORD=***
export OS_TENANT_ID=***
export OS_TENANT_NAME=***
export OS_AUTH_URL=***
export OS_REGION_NAME=***
```


### Mount

You can run swiftfs command with container-name and mountpoint.

```shell
$ swiftfs CONTAINER-NAME MOUNTPOINT
```

### Unmount

Also, you can use fusermount command.

```shell
$ fusermount -u MOUNTPOINT
```

### Options

Print out a option list with "-h" option.

**--debug**

Output debug information

**--no-daemon**

Start an swiftfs process as a foreground (for debugging)

**--logfile, -l**

The logfile name that appends some information instead of stdout/stderr

**----object-cache-time**

The time(sec) that how long is internal object-list cached. default is -1, it will not be cached.

**--create-container, -c**

Create a container if is not exist


## Todo

- Support chmod/chown functions
- Support HTTP compression(net/http package does not support it)
- Reduce the number of building ObjectList
- Performance inprovement when handle a huge number of objects
- Fix bugs

## License

MIT License

## Author

Hironobu Saitoh
<hiro@hironobu.org>
