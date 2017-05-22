package slog

import (
	"github.com/op/go-logging"
	"log"
	"os"
	"strings"
	"github.com/getsentry/raven-go"
	"bytes"
	"errors"
	"runtime"
	"path"
	"unsafe"
	"time"
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

// :TRICKY: have to fork this code with losses (shouldExcludeErr) because want Params != nil
func CaptureMessageAndWait(message string, tags map[string]string, rec *logging.Record, calldepth int) string {
	client := raven.DefaultClient

	if client == nil {
		return ""
	}

	//if client.shouldExcludeErr(message) {
	//	return ""
	//}

	var fn string
	pc, pathname, line, ok := runtime.Caller(calldepth)
	if ok {
		fn = path.Base(pathname)
	}

	// * aggregation key

	// .fmt is private, f*ck
	//key := rec.fmt

	//key := message
	//if ok {
	//	if f := runtime.FuncForPC(pc); f != nil {
	//		key = fmt.Sprintf("%s:%d:%s", fn, line, f.Name())
	//	}
	//}
	_ = pc

	key := message
	lRec := (*LoggingRecord)(unsafe.Pointer(rec))
	if lRec.fmt != nil {
		key = *lRec.fmt
	}

	packet := raven.NewPacket(message, &raven.Message{key, rec.Args})

	if ok {
		extra := packet.Extra
		extra["filename"] = fn
		extra["lineno"] = line
		extra["pathname"] = pathname
	}

	packet.Level = Record2Level(rec)

	eventID, ch := client.Capture(packet, tags)
	<-ch

	return eventID
}

func CaptureErrorAndWait(message string, tags map[string]string, calldepth int, level raven.Severity) string {
	client := raven.DefaultClient

	if client == nil {
		return ""
	}

	//if client.shouldExcludeErr(err.Error()) {
	//	return ""
	//}

	// :TRICKY: original CaptureError() use Exception type, which needs proper error type,
	// but we do not have it for go-logging and log packages

	// aggregating is done by stacktrace

	stacktrace := raven.NewStacktrace(calldepth, 3, client.IncludePaths())
	packet := raven.NewPacket(message, stacktrace)

	packet.Level = level

	eventID, ch := client.Capture(packet, tags)
	<-ch

	return eventID
}


func (l *SentryBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	if level <= logging.WARNING {
		cd := calldepth+2

		//s := rec.Formatted(calldepth+2)
		buf := new(bytes.Buffer)
		logging.DefaultFormatter.Format(cd, rec, buf)
		s := buf.String()
		//fmt.Println(s)

		var tags map[string]string
		if rec.Module != "" {
			tags = map[string]string{
				"module": rec.Module,
			}
		}

		isWarning := level == logging.WARNING
		if isWarning {
			//SendToSentry(s, tags, isWarning)
			CaptureMessageAndWait(s, tags, rec, cd)
		} else {
			//SendToSentry(s, tags, isWarning)
			CaptureErrorAndWait(s, tags, cd, Record2Level(rec))
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
	CaptureErrorAndWait(s, nil, 4, raven.ERROR)

	return n, err
}

func HookStandardLog() {
	log.SetOutput(&SentryLog{})
}
