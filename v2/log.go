package slog

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
	"time"
	"unsafe"

	raven "github.com/getsentry/raven-go"
	"github.com/muravjov/slog/base"
	"github.com/muravjov/slog/sentry"
	logging "github.com/op/go-logging"
)

type SentryBackend struct {
	// we use DefaultClient and global raven.SetDSN()
	//Client *raven.Client
}

type LoggingRecord struct {
	ID     uint64
	Time   time.Time
	Module string
	Level  logging.Level
	Args   []interface{}

	// message is kept as a pointer to have shallow copies update this once
	// needed.
	message   *string
	fmt       *string
	formatter logging.Formatter
	formatted string
}

func Record2Level(rec *logging.Record) raven.Severity {
	res := raven.ERROR
	switch rec.Level {
	case logging.WARNING:
		res = raven.WARNING
	case logging.CRITICAL:
		res = raven.FATAL
	}

	return res
}

func (l *SentryBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	if (level <= logging.WARNING) && (rec.Module != "sentry.errors") {
		cd := calldepth + 2

		//message := rec.Formatted(calldepth+2)
		buf := new(bytes.Buffer)
		logging.DefaultFormatter.Format(cd, rec, buf)
		message := buf.String()
		//fmt.Println(message)

		var tags map[string]string
		if rec.Module != "" {
			tags = map[string]string{
				"module": rec.Module,
			}
		}

		isWarning := level == logging.WARNING

		if isWarning {
			// * aggregation key

			// .fmt is private, f*ck
			//key := rec.fmt

			key := message
			lRec := (*LoggingRecord)(unsafe.Pointer(rec))
			if lRec.fmt != nil {
				key = *lRec.fmt
			}

			sentry.CaptureMessageAndWait(message, tags, cd, &raven.Message{
				Message: key,
				Params:  rec.Args,
			})
		} else {
			sentry.CaptureErrorAndWait(message, tags, cd, Record2Level(rec))
		}
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
	Writer io.Writer
}

// to emulate standard logger
//var std = log.New(os.Stderr, "", log.LstdFlags)
//var std = log.New(os.Stderr, "", 0)

func SkipSpace(s string) string {
	idx := strings.IndexRune(s, ' ')
	if idx != -1 {
		s = s[idx+1:]
	}
	return s
}

// io.Writer interface for log
func (w *SentryLog) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)

	s := string(p)
	// because of log.LstdFlags we need to skip 2 spaces
	s = SkipSpace(SkipSpace(s))
	sentry.CaptureErrorAndWait(s, nil, 4, raven.ERROR)

	return n, err
}

func HookStandardLog(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}

	log.SetOutput(&SentryLog{
		Writer: w,
	})
}

// just like Python' raven
var sentryLogger = logging.MustGetLogger("sentry.errors")

func SetupSentryLogger() {
	sentry.SetSEH(func(err error) {
		sentryLogger.Error(err)
	})
}

func MustSetDSNAndHandler(dsn string) {
	sentry.MustSetDSN(dsn)
	SetupSentryLogger()
}

func RedirectStandardLog(logWriter io.Writer, withSentry bool) {
	if withSentry {
		HookStandardLog(logWriter)
	} else {
		if logWriter != nil {
			log.SetOutput(logWriter)
		}
	}
}

// andStandardLog - hook https://golang.org/pkg/log calls also
func SetupGoLogging(logPath string, dsn string, andStandardLog bool) {
	var logWriter io.Writer = os.Stderr
	if logPath != "" {
		logWriter = base.OpenLog(logPath)
	}

	// *
	// we use go-logging formatter
	//var flag int = log.LstdFlags
	var flag int = 0
	fileBackend := logging.NewLogBackend(logWriter, "", flag)

	logBackends := []logging.Backend{
		fileBackend,
	}

	// *
	withSentry := dsn != ""
	if withSentry {
		MustSetDSNAndHandler(dsn)

		logBackends = append(logBackends, NewSB())
	}

	logging.SetBackend(logBackends...)
	// time formatter is rfc3339Milli = "2006-01-02T15:04:05.999Z07:00" by default,
	// not time.RFC3339 = "2006-01-02T15:04:05Z07:00"
	logging.SetFormatter(logging.MustStringFormatter(
		`%{time} %{level:.4s} %{message}`,
	))

	if andStandardLog {
		RedirectStandardLog(logWriter, withSentry)
	}
}
