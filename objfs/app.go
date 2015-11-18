package objfs

import (
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"

	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func Run() {
	app := cli.NewApp()

	app.Name = "objfs"
	app.Usage = "The file system to mount OpenStack Swift object storage via FUSE."
	app.Version = "0.1alpha"
	app.HideHelp = true
	app.Author = "Hironobu Saitoh"
	app.Email = "hiro@hironobu.org"
	app.ArgsUsage = "container-name mountpoint"

	config := NewConfig()
	defer config.Logfile.Close()

	app.Flags = config.GetFlags()

	app.Action = func(c *cli.Context) {
		if c.Bool("help") || len(c.Args()) < 2 {
			cli.ShowAppHelp(c)
			return
		}

		var err error
		if err = config.SetConfigFromContext(c); err != nil {
			log.Warnf("%v", err)
			return
		}

		if !config.NoDaemon {
			if err = daemonize(c, config); err != nil {
				log.Warnf("%v", err)
				return
			}
		}

		log.Debug("Create a filesystem")
		fs := NewFileSystem(config.Driver, config.MountPoint)

		log.Debug("Mount a filesystem")
		server, err := fs.Mount()
		if err != nil {
			log.Warnf("%v", err)

			if !config.NoDaemon {
				afterDaemonize()
			}
			return
		}

		log.Debugf("ObjFS process with pid %d started", syscall.Getpid())

		if !config.NoDaemon {
			afterDaemonize()
		}

		server.Serve()

		log.Debug("Shutdown")
	}

	app.RunAndExitOnError()
}

func lockfile() string {
	return path.Join(os.TempDir(), ".objfs-startup")
}

func daemonize(c *cli.Context, config *Config) (err error) {
	// TODO: これでOK?
	if _, err = os.Stat(lockfile()); err == nil {
		// Child process
		return nil
	}

	// Create a lockfile
	log.Debug("Create a lockfile")
	f, err := os.OpenFile(lockfile(), os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	f.Close()

	// Spawn a daemon process
	log.Debug("Spawn a daemon process")
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Start()

	// Wait 10 seconds for lunching a child process.
	i := 0
	ok := false
	for i < 10 {
		if _, err = os.Stat(lockfile()); err != nil {
			// Child process has been started successfully
			ok = true
			break
		}

		log.Debug("Wait starting a daemon process")

		time.Sleep(500 * time.Millisecond)
		i++
	}

	// Exit parent process
	if ok {
		os.Exit(0)
	} else {
		log.Warn("ObjFs daemon failed to start")
		os.Exit(1)
	}
	return nil
}

func afterDaemonize() {
	// Ignore SIGCHLD signal
	signal.Ignore(syscall.SIGCHLD)

	// Close STDOUT, STDIN, STDERR
	syscall.Close(0)
	syscall.Close(1)
	syscall.Close(2)

	// Become the process group leader
	syscall.Setsid()

	// Clear umask
	syscall.Umask(022)

	// chdir for root directory
	syscall.Chdir("/")

	// Remove lock file
	log.Debug("Delete a lockfile")
	os.Remove(lockfile())
}
