package main

import (
	"os"
	"time"
	"log"
	"fmt"
	"syscall"
	"slog"
	"io"
)

func main() {
	if false {
		env := os.Environ()

		f := []*os.File{
			os.Stderr,    // (0) stdin
			os.Stdout,   // (1) stdout
			//os.Stderr,   // (2) stderr
		}

		attr := &os.ProcAttr{
			//Dir:   d.WorkDir,
			Env:   env,
			Files: f,
			Sys: &syscall.SysProcAttr{
				//Chroot:     d.Chroot,
				//Credential: d.Credential,
				//Setsid:     true,
			},
		}

		p, err := os.StartProcess("/usr/bin/env", []string{"env"}, attr)
		if err != nil {
			log.Fatalf("Can't start watcher: ", err)
		}

		fmt.Println(p.Pid)
		os.Exit(0)
	}

	if false {
		dsn := os.Args[1]

		slog.MustSetDSN(dsn)

		// :TODO!!!: os.Stderr
		var out io.Writer = os.Stdout
		slog.ProcessStream(os.Stderr, out)
		//slog.ProcessStream(os.Stdin, out)
		os.Exit(0)
	}

	dsn := os.Args[1]
	slog.StartWatcher(dsn, "/tmp/watcher")

	time.Sleep(1 * time.Second)
	slog.UncontrolledCrash()
	time.Sleep(1 * time.Second)
	//_ = time.Sleep
}
