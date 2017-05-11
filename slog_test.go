package slog

import (
	"time"
	"os"
	"testing"
	"github.com/op/go-logging"
	"log"
	"github.com/getsentry/raven-go"
)

func TestSlog(t *testing.T) {
	if false {
		dsn := os.Args[1]
		raven.SetDSN(dsn)

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

		logger.Errorf("error: %s", "arg2")
	}

	if false {
		log.Println("msg1")

		HookStandardLog()

		log.Println("msg2")
		log.Fatal("fatal msg")
	}
}
