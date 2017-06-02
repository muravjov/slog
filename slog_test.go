package slog

import (
	"time"
	"os"
	"testing"
	"github.com/op/go-logging"
	"log"
	"github.com/getsentry/raven-go"
	"math/rand"
	"fmt"
	"bytes"
	"io/ioutil"
	"io"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var seedDone = false

func RandStringBytes(n int) string {
	if !seedDone {
		seedDone = true
		rand.Seed(time.Now().UnixNano())
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestSlog(t *testing.T) {
	dsn := os.Args[2]
	MustSetDSN(dsn)


	if false {
		//go ForceException()
		ForceException()

		raven.CapturePanic(func() {
			ForceException()
		}, nil)

		time.Sleep(time.Hour)
	}

	if false {
		logger := logging.MustGetLogger("example")

		logger.Errorf("error: %s", "arg")

		//backend := logging.NewLogBackend(os.Stdout, "prefix", 0)
		backend := NewSB()
		logging.SetBackend(backend)

		// :TRICKY: stacktrace aggregation = frames aggregation is being done by
		// function and if sources exists locally, by context_line => so
		// changing line (e.g. adding space character) will break aggregation

		// and without sources errors in the same functions will be aggregated
		// :TODO: append Message interface like for CaptureMessageAndWait()

		logger.Errorf("error - %s", RandStringBytes(8))

		logger.Errorf("another error - %s",
			RandStringBytes(8))
	}

	if false {
		log.Println("msg1")

		HookStandardLog()

		//log.Println("msg2")

		log.Fatal("fatal msg: ", RandStringBytes(8))
	}

	if false {
		// :REFACTOR:
		logger := logging.MustGetLogger("example")
		backend := NewSB()
		logging.SetBackend(backend)

		//logger.Warningf("warning: %s", RandStringBytes(8))

		//logger.Warning("static message")
		//logger.Fatal("fatal message")

		type Status struct {
			Vhost  string `protobuf:"bytes,1,opt,name=vhost" json:"vhost,omitempty"`
			Node   string `protobuf:"bytes,2,opt,name=node" json:"node,omitempty"`
			Status string `protobuf:"bytes,3,opt,name=status" json:"status,omitempty"`
			Error  string `protobuf:"bytes,4,opt,name=error" json:"error,omitempty"`
		}

		logger.Errorf("Can't post status: %s, %+v", "502", Status{
			Vhost: "test.ru",
			Node: "test-node",
			Status: "failed",
			Error: RandStringBytes(8),
		})
	}

	if false {
		fmt.Println(RandStringBytes(8))
		fmt.Println(RandStringBytes(8))
	}

	if true {
		stdErr := `
panic: runtime error: integer divide by zero

goroutine 26 [running]:
panic(0x978840, 0xc420010140)
        /home/ilya/opt/programming/golang/git/src/runtime/panic.go:500 +0x1a1
slog.ForceException()
        /home/ilya/opt/programming/g-core/cdn-tools/src/slog/slog.go:19 +0x2c
main.main.func1.1()
        /home/ilya/opt/programming/g-core/cdn-tools/src/mapi/main.go:166 +0x14
created by main.main.func1
        /home/ilya/opt/programming/g-core/cdn-tools/src/mapi/main.go:167 +0xc8
`
		in := bytes.NewBufferString(stdErr)

		ProcessStream(in)
	}
}

// :REFACTOR:
func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

func Assert(b bool)  {
	if !b {
		panic("Assertion error")
	}
}
