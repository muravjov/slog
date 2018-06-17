package main

import (
	"fmt"
	"io"
	"net/http"
	urlModule "net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"

	"github.com/G-Core/slog/base"
	"github.com/bradfitz/iter"
	"github.com/davecgh/go-spew/spew"
)

type RequestContext struct {
	Client *http.Client
	Stats  map[string]int64

	IncrStat func(msg string)

	WaitStatsReady func()
}

func NewRequestContext() *RequestContext {
	rc := &RequestContext{}

	client := &http.Client{}
	rc.Client = client
	//client.Timeout = time.Second * 5

	// for the sake of simplicity, let's choose sync.Map
	// https://medium.com/@deckarep/the-new-kid-in-town-gos-sync-map-de24a6bf7c2c
	//var stats sync.Map
	//
	// No, unnecessarily difficult for now,- go for channels
	stats := make(map[string]int64)
	rc.Stats = stats

	statChan := make(chan string, 1000)
	statDone := NewEvent()
	go func() {
		for msg := range statChan {
			counter, ok := stats[msg]
			if ok {
				counter = counter + 1
			} else {
				counter = 1
			}
			stats[msg] = counter
		}
		statDone.Done()
	}()

	rc.IncrStat = func(msg string) {
		statChan <- msg
	}

	rc.WaitStatsReady = func() {
		close(statChan)
		statDone.Wait()
	}

	return rc
}

func ExecuteGet(url string, rc *RequestContext) {
	var body io.Reader = nil
	req, err := http.NewRequest("GET", url, body)
	base.CheckError(err)

	//req.Header.Add("If-None-Match", `W/"wyzzy"`)
	resp, err := rc.Client.Do(req)
	if err != nil {
		// client.Do() wraps real error with "method: url: <real error>",
		// it is inconveniently
		switch errTyped := err.(type) {
		case *urlModule.Error:
			err = errTyped.Err
		}

		rc.IncrStat(err.Error())
		return
	}
	defer resp.Body.Close()

	//fmt.Println(resp.StatusCode)
	//bdat, err := ioutil.ReadAll(resp.Body)
	//base.CheckError(err)
	//fmt.Println(string(bdat))
	switch dig := resp.StatusCode / 100; dig {
	case 2, 4, 5:
		rc.IncrStat(fmt.Sprintf("%dxx", dig))
	default:
		rc.IncrStat(resp.Status)
	}
}

// .Done() and .Wait() may be implemented via channels like so:
// jobsEnded := make(chan struct{})
// close(jobsEnded) // = .Done()
// <-jobsEnded      // = .Wait()
//
// But channels may be select-ed and wg's - not.
// So, if you do not use select then WaitGroup is just a sufficient match
func NewEvent() *sync.WaitGroup {
	var doneEvent sync.WaitGroup
	doneEvent.Add(1)
	return &doneEvent
}

type JobContext struct {
	StartJob func(jobFunc func())
	Wait     func()
}

func NewJobContext() *JobContext {
	// we need a separate goroutine to account how
	// many jobs are done, as far as possible (to release their goroutine on sending over channel)
	// as we understand that all jobs already started (by signal),
	// then it remains to wait over started-ended and transfer control back
	started := 0
	ended := 0
	endedChan := make(chan struct{})

	jobsStartedChan := make(chan struct{})

	jobsEnded := NewEvent()

	// :TODO: channels are fine and universal, but implementation via sync.Mutex
	// would be more suitable, clearer and simpler - and doesn't need helper goroutines
	go func() {
	mainloop:
		for {
			select {
			case <-endedChan:
				ended = ended + 1
				//fmt.Println("ended: collected")
			case <-jobsStartedChan:
				break mainloop
			}
		}

		cnt := started - ended
		base.Assert(cnt >= 0)
		for range iter.N(cnt) {
			_, ok := <-endedChan
			base.Assert(ok)
			//fmt.Println("ended: collected2")
		}
		close(endedChan)

		jobsEnded.Done()
	}()

	startedChan := make(chan struct{}, 1000)
	go func() {
		for range startedChan {
			started = started + 1
		}
		close(jobsStartedChan)
	}()

	StartJob := func(jobFunc func()) {
		go func() {
			defer func() {
				endedChan <- struct{}{}
				//fmt.Println("ended")
			}()
			jobFunc()
		}()

		// for concurrent job start
		// we accumulate "started" via goroutine and stop it via channel' close()
		// like in RequestContext.WaitStatsReady()
		//started = started + 1
		startedChan <- struct{}{}
		//fmt.Println("started")
	}

	Wait := func() {
		close(startedChan)
		jobsEnded.Wait()
	}

	return &JobContext{
		StartJob: StartJob,
		Wait:     Wait,
	}
}

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

	if true {
		rc := NewRequestContext()
		url := "http://localhost:8789/hello"

		jc := NewJobContext()

		interruptChan := make(chan os.Signal, 1)
		signal.Notify(interruptChan, syscall.SIGINT)

	forCycle:
		for {
			select {
			case <-interruptChan:
				break forCycle
			default:
				jc.StartJob(func() {
					ExecuteGet(url, rc)
				})
			}
		}

		//time.Sleep(time.Second * 5)

		jc.Wait()

		rc.WaitStatsReady()
		spew.Dump(rc.Stats)
	}
}
