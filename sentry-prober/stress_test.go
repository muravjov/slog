package main

import (
	"fmt"
	"testing"

	"github.com/G-Core/slog/base"
	"github.com/bradfitz/iter"
	"github.com/davecgh/go-spew/spew"
)

func TestStress(t *testing.T) {
	if false {
		rc := NewRequestContext()

		//url := "http://example.com"
		//url := "http://localhost:9000/organizations/sentry/stats/"
		url := "http://localhost:8789/hello"

		ExecuteGet(url, rc)

		rc.WaitStatsReady()
		spew.Dump(rc.Stats)
	}

	if false {
		rc := NewRequestContext()
		url := "http://localhost:8789/hello"

		jc := NewJobContext()

		for range iter.N(5) {
			jc.StartJob(func() {
				ExecuteGet(url, rc)
			})
		}

		//time.Sleep(time.Second * 5)

		jc.Wait()

		rc.WaitStatsReady()
		spew.Dump(rc.Stats)
	}

	if false {
		url := "http://localhost:8789/hello"
		_ = url
		rc := NewRequestContext()

		jobFunc := func() {
			ExecuteGet(url, rc)
			fmt.Println("ggg")
		}

		rps := 10.     // 10 requests per seconds
		duration := 2. // seconds

		///
		timeElapsed := MakeStress(jobFunc, rps, duration)

		rc.WaitStatsReady()
		//spew.Dump(rc.Stats)
		PrintReport(rc.Stats, timeElapsed, 10)
	}

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
		rc := NewRequestContext()

		eventID, err := PostSentryEvent(event, client, rc)
		base.CheckError(err)
		fmt.Println(eventID)

		rc.WaitStatsReady()
		spew.Dump(rc.Stats)
	}

	if false {
		stats := StatsType{
			"1": 1,
			"2": 2,
			"3": 3,
			"4": 4,
			"5": 5,
			"6": 6,
		}
		var timeElapsed float64 = 1 // seconds

		aggregationCount := 5 // все, что свыше => в "Other Stats"

		PrintReport(stats, timeElapsed, aggregationCount)
	}
}
