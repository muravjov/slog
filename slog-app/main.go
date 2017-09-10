package main

import (
	"github.com/G-Core/slog"
	flag "github.com/spf13/pflag"
	"net/http"
	"time"
)

func main() {
	//if false {
	//	env := os.Environ()
	//
	//	f := []*os.File{
	//		os.Stderr,    // (0) stdin
	//		os.Stdout,   // (1) stdout
	//		//os.Stderr,   // (2) stderr
	//	}
	//
	//	attr := &os.ProcAttr{
	//		//Dir:   d.WorkDir,
	//		Env:   env,
	//		Files: f,
	//		Sys: &syscall.SysProcAttr{
	//			//Chroot:     d.Chroot,
	//			//Credential: d.Credential,
	//			//Setsid:     true,
	//		},
	//	}
	//
	//	p, err := os.StartProcess("/usr/bin/env", []string{"env"}, attr)
	//	if err != nil {
	//		log.Fatalf("Can't start watcher: ", err)
	//	}
	//
	//	fmt.Println(p.Pid)
	//	os.Exit(0)
	//}
	//
	//if false {
	//	dsn := os.Args[1]
	//
	//	slog.MustSetDSN(dsn)
	//
	//	slog.ProcessStream(os.Stderr)
	//	//slog.ProcessStream(os.Stdin, out)
	//	os.Exit(0)
	//}

	sleepFail := flag.BoolP("sleep-fail", "", false, "sleep and fail instead of http server")
	watcherStderr := flag.StringP("watcher-stderr", "", "", "where to redirect stderr")

	flag.Parse()

	dsn := flag.Arg(0) //os.Args[1]
	//errFileName := "/tmp/watcher"
	//errFileName := ""
	errFileName := *watcherStderr

	slog.StartWatcher(dsn, errFileName)

	if *sleepFail {
		time.Sleep(1 * time.Second)
		slog.UncontrolledCrash()
		time.Sleep(1 * time.Second)
		//_ = time.Sleep
	} else {
		handler := func(w http.ResponseWriter, r *http.Request) {
			slog.UncontrolledCrash()
		}

		http.HandleFunc("/", handler)
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			panic(err)
		}
	}
}
