package slog

import (
	"github.com/op/go-logging"
	"log"
	"os"
	"strings"
	"github.com/getsentry/raven-go"
	"bytes"
	"errors"
)

func ForceException() {
	i := 0
	i = 1 / i
}

type SentryBackend struct {
	// we use DefaultClient and global raven.SetDSN()
	//Client *raven.Client
}

func SendToSentry(s string, tags map[string]string, isWarning bool) {
	// Capture vs Capture*AndWait
	// we prefer wait to safely submit errors
	if isWarning {
		// without stacktrace
		raven.CaptureMessageAndWait(s, tags)
	} else {
		raven.CaptureErrorAndWait(errors.New(s), tags)
	}
}

func (l *SentryBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	if level <= logging.WARNING {
		//s := rec.Formatted(calldepth+2)
		buf := new(bytes.Buffer)
		logging.DefaultFormatter.Format(calldepth+2, rec, buf)
		s := buf.String()
		//fmt.Println(s)

		var tags map[string]string
		if rec.Module != "" {
			tags = map[string]string{
				"module": rec.Module,
			}
		}

		SendToSentry(s, tags, level == logging.WARNING)
	}
	return nil
}

//
// :TRICKY: we want LeveledBackend interface to force level WARNING
//

func (l *SentryBackend) GetLevel(module string) logging.Level {
	return logging.WARNING
}

func (l *SentryBackend) SetLevel(level logging.Level, module string) {
}

func (l *SentryBackend) IsEnabledFor(level logging.Level, module string) bool {
	return level <= logging.WARNING
}

func NewSB() logging.LeveledBackend {
	return &SentryBackend{}
}

//
// log
//

type SentryLog struct {
}

// to emulate standard logger
//var std = log.New(os.Stderr, "", log.LstdFlags)
var std = log.New(os.Stderr, "", 0)

func SkipSpace(s string) string {
	idx := strings.IndexRune(s, ' ')
	if idx != -1 {
		s = s[idx+1:]
	}
	return s
}

// io.Writer interface for log
func (w *SentryLog) Write(p []byte) (n int, err error) {
	n, err = os.Stderr.Write(p)

	s := string(p)
	// because of log.LstdFlags we need to skip 2 spaces
	s = SkipSpace(SkipSpace(s))
	//SendToSentry(s, nil, false)

	return n, err
}

func HookStandardLog() {
	log.SetOutput(&SentryLog{})
}
