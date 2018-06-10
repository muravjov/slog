package base

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/G-Core/slog/sentry"
	raven "github.com/getsentry/raven-go"
	"github.com/maruel/panicparse/stack"
)

func ProcessStream(in io.Reader, watcheePid int, watcheeArgs []string) {
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
		// :TRICKY: that goes to os.Stderr like in WatchReader.Read()
		log.Printf("Post-mortem detected, %v, pid=%d", watcheeArgs, watcheePid)

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

		msg := fmt.Sprintf("Post-mortem %v, pid=%d: %s", watcheeArgs, watcheePid, accOut)
		sentry.CaptureAndWait(sentry.Interface2Packet(msg, stacktrace, raven.FATAL), nil)
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
