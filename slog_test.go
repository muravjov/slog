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
	raven.SetDSN(dsn)


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

	if true {
		// :REFACTOR:
		logger := logging.MustGetLogger("example")
		backend := NewSB()
		logging.SetBackend(backend)

		//logger.Warningf("warning: %s", RandStringBytes(8))

		//logger.Warning("static message")
		logger.Fatal("fatal message")
	}

	if false {
		fmt.Println(RandStringBytes(8))
		fmt.Println(RandStringBytes(8))
	}

}
