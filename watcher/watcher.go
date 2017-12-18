/*
Package watcher implements StartWatcher() function to catch Golang panics with a special watchdog process.
*/
package watcher

import (
	"reflect"
	"unsafe"
	"os"
	"os/signal"
	"syscall"
	"log"
	"fmt"
	"github.com/erikdubbelboer/gspt"
	"github.com/getsentry/raven-go"
	"io"
	"bytes"
	"github.com/kardianos/osext"
	"github.com/maruel/panicparse/stack"
	"io/ioutil"
	"github.com/G-Core/slog/sentry"
)

// it's hack,
// use gspt.SetProcTitle()
func setProcessName(name string) {
	argv0str := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[0]))
	argv0 := (*[1 << 30]byte)(unsafe.Pointer(argv0str.Data))[:argv0str.Len]

	n := copy(argv0, name)
	if n < len(argv0) {
		argv0[n] = 0
	}
}

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

		s := fmt.Sprintf("Go watcher for pid: %d", WatcheePid())
		//log.Print(s)

		//os.Args[0] = fmt.Sprintf("Go watcher for pid: %d", os.Getppid())
		//setProcessName(s)
		gspt.SetProcTitle(s)

		ProcessStream(os.Stdin)

		os.Exit(0)
	}
}


func CheckFatal(format string, err error) {
	if err != nil {
		log.Fatalf(format, err)
	}
}

func OpenLog(errFileName string) *os.File {
	logFile, err := os.OpenFile(errFileName,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(0640))
	CheckFatal("Can't open: %s", err)
	return logFile
}

// Starts a watchdog process to catch panics and to store them into file errFileName and
// Sentry
func StartWatcher(dsn string, errFileName string) {
	cx, err := osext.Executable()
	CheckFatal("osext.Executable(): %s", err)

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
		logFile = OpenLog(errFileName)
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
	CheckFatal("os.Pipe(): %s", err)

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
	CheckFatal("Can't start watcher: %s", err)

	in.Close()
	// redirect stderr to watcher stdin
	syscall.Dup2(int(wpipe.Fd()), 2)
}

func ProcessStream(in io.Reader) {
	// :TRICKY: stack.ParseDump() searches for
	//    goroutine <N> [<status>]:
	// but every crash starts like that:
	//    panic: <real err>\n
	// see printpanics()

	// :TODO: for debugging purposes add prints if _SLOG_WATCHER_DEBUG=true
	// e.g. systemd kills all processes by default, by KillMode=control-group
	//fmt.Fprintln(os.Stderr, "ProcessStream1")

	wr := NewWR(in)
	goroutines, err := stack.ParseDump(wr, ioutil.Discard)
	if err != nil {
		log.Fatalf("ParseDump: %s", err)
	}

	if len(goroutines) != 0 {
		wp := WatcheePid()
		// :TRICKY: that goes to os.Stderr like in WatchReader.Read()
		log.Printf("Post-mortem detected, %v, pid=%d", os.Args, wp)

		failedG := goroutines[0]
		//fmt.Println(failedG)

		calls := failedG.Stack.Calls
		var frames []*raven.StacktraceFrame
		for i := range calls {
			call := calls[len(calls)-1-i]

			f := call.Func
			// NewStacktraceFrame
			frame := &raven.StacktraceFrame{
				Filename: call.SourcePath,
				Function: f.Name(),
				Module:   f.PkgName(),

				AbsolutePath: call.SourcePath,
				Lineno:       call.Line,
				InApp:        false,
			}

			frames = append(frames, frame)
		}

		stacktrace := &raven.Stacktrace{Frames: frames}

		accOut := wr.Buf.String()

		sentry.CaptureAndWait(sentry.Interface2Packet(fmt.Sprintf("Post-mortem %v, pid=%d: %s", os.Args, wp, accOut), stacktrace, raven.FATAL), nil)
	}
}

//type DoubleWriter struct {
//	origWriter io.Writer
//	Buf        *bytes.Buffer
//}
//
//func NewDW(out io.Writer) *DoubleWriter {
//	return &DoubleWriter{
//		origWriter: out,
//		Buf: bytes.NewBuffer(nil),
//	}
//}
//
//func (dw *DoubleWriter) Write(p []byte) (int, error) {
//	n, err := dw.origWriter.Write(p)
//	dw.Buf.Write(p)
//
//	return n, err
//}

type WatchReader struct {
	origReader io.Reader
	Buf        *bytes.Buffer
}

func NewWR(in io.Reader) *WatchReader {
	return &WatchReader{
		origReader: in,
		Buf:        bytes.NewBuffer(nil),
	}
}

func (wr *WatchReader) Read(p []byte) (int, error) {
	n, err := wr.origReader.Read(p)
	if n > 0 {
		dat := p[:n]
		os.Stderr.Write(dat)
		wr.Buf.Write(dat)
	}
	return n, err
}
