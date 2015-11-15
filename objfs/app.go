package objfs

import (
	"os"
	"os/exec"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func NewApp() *cli.App {
	app := cli.NewApp()

	app.Name = "objfs"
	app.Usage = "The file system to mount OpenStack Swift object storage via FUSE."
	app.Version = "0.1alpha"
	app.HideHelp = true
	app.Author = "Hironobu Saitoh"
	app.Email = "hiro@hironobu.org"

	config := NewConfig()
	app.Flags = config.GetFlags()

	app.Before = func(c *cli.Context) (err error) {
		return nil
	}

	app.Action = func(c *cli.Context) {
		if c.Bool("help") || len(c.Args()) == 0 {
			cli.ShowAppHelp(c)
			return
		}

		var err error
		if err = config.SetConfigFromContext(c); err != nil {
			log.Warnf("%v", err)
			return
		}

		if err = daemonize(c, config); err != nil {
			log.Warnf("%v", err)
			return
		}

		fs := NewFileSystem(config.Driver, config.MountPoint)
		if err = fs.Mount(); err != nil {
			log.Warnf("%v", err)
		}
	}

	return app
}

func daemonize(c *cli.Context, config *Config) (err error) {

	// logfile
	var logfile *os.File
	if config.Logfile != "" {
		logfile, err = os.OpenFile(config.Logfile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
	}
	defer logfile.Close()

	if !c.Bool("daemon") {
		args := []string{
			"--daemon",
		}
		args = append(args, os.Args[1:]...)

		cmd := exec.Command(os.Args[0], args...)
		cmd.Start()
		os.Exit(0)

	} else {
		// Write some outputs to the logfile if provided
		if logfile != nil {
			log.SetFormatter(&log.TextFormatter{
				DisableColors:    true,
				DisableTimestamp: true,
				DisableSorting:   true,
			})

			syscall.Dup2(int(logfile.Fd()), 1)
			syscall.Dup2(int(logfile.Fd()), 2)

			log.SetOutput(logfile)
		}

		// close STDOUT, STDIN, STDERR
		syscall.CloseOnExec(0)
		syscall.CloseOnExec(1)
		syscall.CloseOnExec(2)

		// become the process group leader
		syscall.Setsid()

		// clear umask
		syscall.Umask(022)

		// chdir for root directory
		syscall.Chdir("/")
	}
	return nil
}
