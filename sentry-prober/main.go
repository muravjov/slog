package main

import (
	flag "github.com/spf13/pflag"
	"log"
	"github.com/G-Core/slog/sentry"
	"github.com/G-Core/slog"
	"github.com/getsentry/raven-go"
	"fmt"
	"github.com/G-Core/slog/util"
	"github.com/G-Core/slog/watcher"
)

func main() {
	isWarning := flag.BoolP("warning", "", false, "warning (without stacktrace) vs error")
	pMessage := flag.StringP("message", "", "sentry-prober", "message to send")
	isRandom := flag.BoolP("random", "", false, "add random string to message")
	isWatcher := flag.BoolP("watcher", "", false, "send message via watcher")

	flag.Parse()

	// :TODO: sentry-prober --help to show required argument: DSN
	dsn := flag.Arg(0)
	if dsn == "" {
		log.Fatalf("DSN argument required")
	}
	slog.MustSetDSNAndHandler(dsn)

	message := *pMessage

	key := message
	var args []interface{} = nil
	if *isRandom {
		arg := util.RandStringBytes(8)
		args = []interface{}{
			arg,
		}
		message = fmt.Sprintf(key, arg)
	}

	if *isWatcher {
		watcher.StartWatcher(dsn, "")

		fmt.Printf(`Do exception to invoke watcher to send post-mortem
`)

		slog.ForceException()
	} else {
		fmt.Printf(`Sending "%s"
`, message)

		if *isWarning {
			sentry.CaptureMessageAndWait(message, nil, 0, &raven.Message{
				Message: key,
				Params: args,
			})
		} else {
			sentry.CaptureErrorAndWait(message, nil, 0, raven.ERROR)
		}
	}
}
