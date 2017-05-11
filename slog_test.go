package slog

import (
	"github.com/getsentry/raven-go"
	"time"
	"os"
	"testing"
)

func ForceException() {
	i := 0
	i = 1 / i
}

func TestSlog(t *testing.T) {
	if true {
		dsn := os.Args[1]
		raven.SetDSN(dsn)

		//go ForceException()
		ForceException()

		raven.CapturePanic(func() {
			ForceException()
		}, nil)

		time.Sleep(time.Hour)
	}
}
