package daemon

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/sevlyar/go-daemon"
)

type Daemon struct {
	pidFile string
	logFile string
	ctx     *daemon.Context
}

var theDaemon *Daemon

func Start(pidFile, logFile string, stopFlag bool, termHandler func(os.Signal) error, mountCallback, serveCallback func()) error {

	var child *os.Process
	var err error

	if termHandler == nil {
		return errors.New("cannot create daemon without handing SIGTERM")
	}

	dctx := &daemon.Context{
		PidFileName: pidFile,
		PidFilePerm: 0644,
		LogFileName: logFile,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
	}

	theDaemon = &Daemon{
		pidFile: pidFile,
		logFile: logFile,
		ctx:     dctx,
	}

	daemon.AddCommand(daemon.BoolFlag(&stopFlag), syscall.SIGTERM, termHandler)

	child, err = theDaemon.ctx.Reborn()
	if err != nil {
		log.Printf("Unable to start. Error: %v", err)
		os.Exit(1)
	}
	if child != nil {
		// in parent
		fmt.Fprintf(os.Stderr, "Check %s for zmount logs\n", theDaemon.logFile)
		return nil
	}

	// --- in child process
	log.Println("Child daemon process has started")
	defer func() {
		err = theDaemon.ctx.Release()
		if err != nil {
			log.Println("Error releasing daemon resources:", err)
		}
	}()
	mountCallback()
	go func() {
		err := daemon.ServeSignals()
		if err != nil {
			log.Printf("Signal error: %v", err)
		}
	}()
	serveCallback()
	return nil
}

func Stop() {
	if len(daemon.ActiveFlags()) > 0 {
		d, err := theDaemon.ctx.Search()
		if err != nil {
			log.Fatalf("Unable send signal to the daemon: %s", err.Error())
		}
		err = daemon.SendCommands(d)
		if err != nil {
			log.Printf("Error sending stop signal %v", err)
		}
	}
}