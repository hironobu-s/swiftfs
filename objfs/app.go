package objfs

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

const (
	DAEMONIZE_STARTING = iota
	DAEMONIZE_SUCCESS
	DAEMONIZE_FAIL
)

func Run() {
	app := cli.NewApp()

	app.Name = APP_NAME
	app.Usage = "The file system to mount OpenStack Swift object storage via FUSE."
	app.Version = APP_VERSION
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

		daemonized := false
		if !config.NoDaemon {
			if err = daemonize(c, config); err != nil {
				log.Warnf("%v", err)
				return
			}
			daemonized = true
		}

		log.Debug("Create a filesystem")
		fs := NewFileSystem(config)

		log.Debug("Mount a filesystem")
		server, err := fs.Mount()
		if err != nil {
			log.Warnf("%v", err)

			if daemonized {
				afterDaemonize(err)
			}
			return
		}

		if daemonized {
			afterDaemonize(nil)
		}

		// main loop
		log.Debugf("ObjFS process with pid %d started", syscall.Getpid())
		server.Serve()

		log.Debug("Shutdown")
	}

	app.RunAndExitOnError()
}

// Spawn a child process and waiting for completing the launch.
func daemonize(c *cli.Context, config *Config) (err error) {
	if config.NoDaemon {
		// child process
		return nil
	}

	// Spawn a daemon process
	log.Debug("Spawn a daemon process")

	args := []string{"--no-daemon"}
	args = append(args, os.Args[1:]...)

	// Used in IPC
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.ExtraFiles = []*os.File{w}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Start()

	// Wait 30 seconds for lunching a child process.
	status := DAEMONIZE_STARTING
	go func() {
		buf := make([]byte, 1)
		r.Read(buf)

		if int(buf[0]) == DAEMONIZE_SUCCESS {
			status = int(buf[0])
		} else {
			status = DAEMONIZE_FAIL
		}
	}()

	i := 0
	for i < 60 {
		if status != DAEMONIZE_STARTING {
			break
		}
		time.Sleep(500 * time.Millisecond)
		i++

		log.Debug("Wait starting a child process")
	}

	// Exit parent process
	if status == DAEMONIZE_SUCCESS {
		log.Debug("objfs started successfuly")
		os.Exit(0)
	} else {
		log.Warn("objfs failed to start")
		os.Exit(1)
	}
	return nil
}

func afterDaemonize(err error) {
	// Ignore SIGCHLD signal
	signal.Ignore(syscall.SIGCHLD)

	// Close STDOUT, STDIN, STDERR
	syscall.Close(0)
	syscall.Close(1)
	syscall.Close(2)

	// Become the process group leader
	syscall.Setsid()

	// // Clear umask
	syscall.Umask(022)

	// // chdir for root directory
	syscall.Chdir("/")

	// Notify that the child process started successfuly
	pipe := os.NewFile(uintptr(3), "pipe")
	if pipe != nil {
		defer pipe.Close()
		if err == nil {
			pipe.Write([]byte{DAEMONIZE_SUCCESS})
		} else {
			pipe.Write([]byte{DAEMONIZE_FAIL})
		}
	}
}
