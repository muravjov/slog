package main

import (
	"fmt"
	"testing"

	"github.com/muravjov/slog"
	"github.com/muravjov/slog/stress"
	"github.com/davecgh/go-spew/spew"
)

func TestStress(t *testing.T) {
	if false {
		// client
		dsn := "http://2b59a34482ac46a68f5a4d6ec79114f8:a9b4e4dedc7447fbbc805f73ea6fc4c3@localhost:9000/2"

		// event
		event := &Event{
			isError:  true,
			key:      "sentry-fds21: %s",
			isRandom: true,
		}

		////////////////////////
		client := MakeClient(dsn)

		//httpClient := &http.Client{}
		rc := stress.NewRequestContext()

		eventID, err := PostSentryEvent(event, client, rc)
		base.CheckError(err)
		fmt.Println(eventID)

		rc.WaitStatsReady()
		spew.Dump(rc.Stats)
	}

}
