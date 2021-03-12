package slog

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/muravjov/slog"
	slogV2 "github.com/muravjov/slog/v2"
	"github.com/muravjov/slog/watcher"
	"github.com/evalphobia/logrus_sentry"
	"github.com/getsentry/raven-go"
	"github.com/op/go-logging"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func ForceException() {
	i := 0
	i = 1 / i
}

// go program may neither avoid crash from this, nor manage it -
// the panic mandatory goes to stderr + os.exit(2)
// see https://github.com/golang/go/issues/20161
func UncontrolledCrash() {
	go func() {
		ForceException()
	}()
}
func OpenLogOrNil(logPath string) io.Writer {
	var logWriter io.Writer
	if logPath != "" {
		logWriter = base.OpenLog(logPath)
	}
	return logWriter
}

func SetupLog(logPath string, dsn string) {
	logWriter := OpenLogOrNil(logPath)

	withSentry := dsn != ""

	if withSentry {
		slogV2.MustSetDSNAndHandler(dsn)
	}
	watcher.StartWatcher(dsn, logPath)

	slogV2.RedirectStandardLog(logWriter, withSentry)
}

func SetupLogrus(logPath string, dsn string) {
	logWriter := OpenLogOrNil(logPath)

	// *
	if logWriter != nil {
		logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})
		logrus.SetOutput(logWriter)
	}

	// *
	if dsn != "" {
		slogV2.MustSetDSNAndHandler(dsn)

		// :TRICKY: with right timeout 5 sec
		//hook, err := logrus_sentry.NewSentryHook(dsn, []logrus.Level{
		hook, err := logrus_sentry.NewWithClientSentryHook(raven.DefaultClient, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
		})
		base.CheckFatal("Can't create logrus_sentry.SentryHook: ", err)

		// :TODO: now Sentry errors are being logged to os.Stderr,- "Failed to fire hook: ..."
		// log to a local file, if needed
		hook.Timeout = time.Second * (5 + 1)
		hook.StacktraceConfiguration.Enable = true

		// :TRICKY: logrus doesn't keep original format and args; instead it does fmt.Sprintf(format, args...)
		// several times, so we have to turn on stackraces for warnings also
		// :TODO: fix upstream:
		// - by correcting func (hook *SentryHook) Fire(entry *logrus.Entry)
		// - making packet like packet := raven.NewPacket(message, &raven.Message{format, args})
		hook.StacktraceConfiguration.Level = logrus.WarnLevel

		logrus.AddHook(hook)

	}
	watcher.StartWatcher(dsn, logPath)
}

func AddForceErrorOption() *string {
	return flag.StringP("force-error", "", "no", "emulate error for logging {no, error, panic}, default = no")
}

func RunForceErrorOption(forceError string, errorFunc func(string)) {
	switch forceError {
	case "no":
	case "error":
		errorFunc("--force-error")
	case "panic":
		ForceException()
	default:
		log.Fatalf("--force-error' bad choice: %s is not in [no, error, panic]", forceError)
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
		slogV2.MustSetDSNAndHandler(dsn)

		logBackends = append(logBackends, slogV2.NewSB())
	}
	watcher.StartWatcher(dsn, logPath)

	logging.SetBackend(logBackends...)
	// time formatter is rfc3339Milli = "2006-01-02T15:04:05.999Z07:00" by default,
	// not time.RFC3339 = "2006-01-02T15:04:05Z07:00"
	logging.SetFormatter(logging.MustStringFormatter(
		`%{time} %{level:.4s} %{message}`,
	))

	if andStandardLog {
		slogV2.RedirectStandardLog(logWriter, withSentry)
	}
}
