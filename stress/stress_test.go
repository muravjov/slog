package stress

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/bradfitz/iter"
	"github.com/davecgh/go-spew/spew"
	"github.com/muravjov/slog/base"
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
		timeElapsed := MakeStress(jobFunc, rps, duration, -1)

		rc.WaitStatsReady()
		//spew.Dump(rc.Stats)
		PrintReport(rc.Stats, timeElapsed, 10)
	}

	if false {
		stats := Statuses{
			"1": 1,
			"2": 2,
			"3": 3,
			"4": 4,
			"5": 5,
			"6": 6,
		}
		var timeElapsed float64 = 1 // seconds
		stressTimes := StressTimes{
			ElapsedTime:      timeElapsed,
			SpawningJobsTime: timeElapsed,
		}

		aggregationCount := 5 // все, что свыше => в "Other Stats"

		PrintReport(&StatsType{Statuses: stats}, stressTimes, aggregationCount)
	}

	if false {
		dsn := "http://aaa:bbb@localhost:9001/2"

		route := NewRoute(`.*`, func(w http.ResponseWriter, r *http.Request, match Match) {
			b := MarshalIndent(map[string]string{})
			ServeJSON(w, http.StatusOK, b)
		})

		ServeDummyHTTP(dsn, route, "")
	}

	if false {
		tlsConfig := NewTLSConfig("tls.toml")
		base.Assert(tlsConfig != nil)
	}

	if false {
		tlsConfig := NewServerTLSConfig("tls.toml")
		base.Assert(tlsConfig != nil)
	}

}
