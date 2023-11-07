package slog

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/muravjov/slog/base"
	logging "github.com/op/go-logging"
)

func TestSlog(t *testing.T) {
	t.SkipNow()

	if true {
		dsn := os.Getenv("SENTRY_DSN")
		SetupGoLogging("", dsn, true)

		logger := logging.MustGetLogger("example")
		logger.Errorf("Error msg: %s", base.RandStringBytes(8))
		logger.Warningf("Warning msg: %s", base.RandStringBytes(8))
		logger.Debugf("Debug msg: %s", base.RandStringBytes(8))

		log.Printf("And std error msg: %s", base.RandStringBytes(8))

		// :KLUDGE: wait for sending sentry events in full
		time.Sleep(time.Second * 15)
	}

}
