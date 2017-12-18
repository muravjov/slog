/*
Package sentry implements sending errors and messages to Sentry with proper aggregation.
Sentry is crash reporting and aggregation platform, https://sentry.io
*/
package sentry

import (
	"github.com/getsentry/raven-go"
	"runtime"
	"path"
	"log"
	"time"
)

// Handler type to hook events when sending to Sentry failed
type SentryErrorHandlerType func(err error)
var SentryErrorHandler SentryErrorHandlerType = nil

// Install handler to hook events when sending to Sentry failed
func SetSEH(seh SentryErrorHandlerType) {
	SentryErrorHandler = seh
}

func CaptureAndWait(packet *raven.Packet, tags map[string]string) string {
	client := raven.DefaultClient

	if client == nil {
		return ""
	}

	//if client.shouldExcludeErr(err.Error()) {
	//	return ""
	//}

	eventID, ch := client.Capture(packet, tags)
	err := <-ch

	if err != nil && SentryErrorHandler != nil {
		SentryErrorHandler(err)
	}

	return eventID
}

func Interface2Packet(message string, iObject raven.Interface, level raven.Severity) *raven.Packet {
	// :TRICKY: original CaptureError() use Exception type, which needs proper error type,
	// but we do not have it for go-logging and log packages
	packet := raven.NewPacket(message, iObject)
	packet.Level = level

	return packet
}

// CaptureErrorAndWait() sends message to Sentry and returns eventID
// Aggregating is done by stacktrace
func CaptureErrorAndWait(message string, tags map[string]string, calldepth int, level raven.Severity) string {
	client := raven.DefaultClient

	if client == nil {
		return ""
	}

	stacktrace := raven.NewStacktrace(calldepth, 3, client.IncludePaths())
	return CaptureAndWait(Interface2Packet(message, stacktrace, level), tags)
}

// CaptureMessageAndWait() sends message to Sentry and returns eventID
// Aggregating is done by iObject.Message attribute
// Additional info is filename and line number
func CaptureMessageAndWait(message string, tags map[string]string, calldepth int, iObject *raven.Message) string {
	packet := Interface2Packet(message, iObject, raven.WARNING)


	var fn string
	pc, pathname, line, ok := runtime.Caller(calldepth)
	if ok {
		fn = path.Base(pathname)
	}
	_ = pc

	if ok {
		extra := packet.Extra
		extra["filename"] = fn
		extra["lineno"] = line
		extra["pathname"] = pathname
	}

	return CaptureAndWait(packet, tags)
}

// = raven.SetDSN(dsn) with checks
func MustSetDSN(dsn string) {
	err := raven.SetDSN(dsn)
	if err != nil {
		log.Fatalf("Bad Sentry DSN '%s': %s", dsn, err)
	}

	// we don't want to get stuck if not working DSN
	// 5 seconds should be enough to send to Sentry
	ht, ok := raven.DefaultClient.Transport.(*raven.HTTPTransport)
	if ok {
		ht.Timeout = time.Second * 5
	}
}
