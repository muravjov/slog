/*
Package watcher implements StartWatcher() function to catch Golang panics with a special watchdog process.
*/
package watcher

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/G-Core/slog/base"
	"github.com/G-Core/slog/sentry"
	"github.com/erikdubbelboer/gspt"
	"github.com/kardianos/osext"
)

// it's hack,
// use gspt.SetProcTitle()
// func setProcessName(name string) {
// 	argv0str := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[0]))
// 	argv0 := (*[1 << 30]byte)(unsafe.Pointer(argv0str.Data))[:argv0str.Len]

// 	n := copy(argv0, name)
// 	if n < len(argv0) {
// 		argv0[n] = 0
// 	}
// }

func WatcheePid() int {
	return os.Getppid()
}

func init() {
	dsn, exists := os.LookupEnv("_SLOG_WATCHER")

	if exists {
		go func() {
			// Do not let systemd or so stop the watcher before event is submitted.

			// https://golang.org/pkg/os/signal/#example_Notify
			c := make(chan os.Signal, 1)
			// A SIGHUP, SIGINT, or SIGTERM signal causes the program to exit =>
			// so handle them.
			signal.Notify(c, os.Interrupt)
			signal.Notify(c, syscall.SIGTERM)
			signal.Notify(c, syscall.SIGHUP)

			for s := range c {
				log.Printf("Watcher ignored signal: %s", s)
			}
		}()

		if dsn != "" {
			sentry.MustSetDSN(dsn)
			// no go-logging dep
			//SetupSentryLogger()
			sentry.SetSEH(func(err error) {
				log.Print(err)
			})
		}

		watcheePid := WatcheePid()
		s := fmt.Sprintf("Go watcher for pid: %d", watcheePid)
		//log.Print(s)

		//os.Args[0] = fmt.Sprintf("Go watcher for pid: %d", os.Getppid())
		//setProcessName(s)
		gspt.SetProcTitle(s)

		base.ProcessStream(os.Stdin, watcheePid, os.Args)

		os.Exit(0)
	}
}

// Starts a watchdog process to catch panics and to store them into file errFileName and
// Sentry
func StartWatcher(dsn string, errFileName string) {
	cx, err := osext.Executable()
	base.CheckFatal("osext.Executable(): %s", err)

	mark := fmt.Sprintf("%s=%s", "_SLOG_WATCHER", dsn)
	env := os.Environ()
	env = append(env, mark)

	var errFile *os.File
	var logFile *os.File
	if errFileName == "" {
		errFile = os.Stderr
		//logFile, err = os.Open(os.DevNull)
		//CheckFatal("Can't open: %s", err)
	} else {
		logFile = base.OpenLog(errFileName)
		errFile = logFile
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	// bad file descriptor
	//in := os.Stderr
	in, wpipe, err := os.Pipe()
	base.CheckFatal("os.Pipe(): %s", err)

	f := []*os.File{
		in,        // (0) stdin
		os.Stdout, // (1) stdout
		errFile,   // (2) stderr
	}

	attr := &os.ProcAttr{
		//Dir:   d.WorkDir,
		Env:   env,
		Files: f,
		Sys:   &syscall.SysProcAttr{
		//Chroot:     d.Chroot,
		//Credential: d.Credential,
		//Setsid:     true,
		},
	}

	_, err = os.StartProcess(cx, os.Args, attr)
	base.CheckFatal("Can't start watcher: %s", err)

	in.Close()
	// redirect stderr to watcher stdin
	syscall.Dup2(int(wpipe.Fd()), 2)
}
