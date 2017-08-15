# Golang logging integration to Sentry

Package slog implements way to log to [Sentry](https://github.com/getsentry/sentry) along with such a logging libraries as standard package *log* and [go-logging](https://github.com/op/go-logging).
Also it provides a way to catch Golang panics with a special watchdog process. For logging to Sentry [raven-go](github.com/getsentry/raven-go) library is used.

# Installation

    go get github.com/github.com/G-Core/slog

# Usage

```golang
dsn := "https://aaa:bbb@app.getsentry.com/nnn"
slog.MustSetDSN(dsn)

// log package support
slog.HookStandardLog()
// to capture panics
slog.StartWatcher(dsn, "")

// you cannot survive that, see 
// https://github.com/golang/go/issues/20161
go func() {
	i := 0
	i = 1 / i
}()

```

After that code you receive an error into Sentry like that:
```
Post-mortem [/usr/sbin/cdn-mapi], pid=1: panic: runtime error: integer divide by zero

goroutine 30488 [running]:
slog.UncontrolledCrash.func1()
	/slog.go:36 +0x11
created by slog.UncontrolledCrash
	/slog.go:37 +0x35
```