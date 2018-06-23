package main

import (
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"
	"time"

	"github.com/G-Core/slog/base"
	"github.com/bradfitz/iter"
)

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

	canStartJob := true
	StartJob := func(jobFunc func()) {
		base.Assert(canStartJob)

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
		canStartJob = false
		close(startedChan)
		jobsEnded.Wait()
	}

	return &JobContext{
		StartJob: StartJob,
		Wait:     Wait,
	}
}

func MakeStress(jobFunc func(), rps float64, duration float64) float64 {
	jc := NewJobContext()

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, syscall.SIGINT)

	// https://stackoverflow.com/questions/19992334/how-to-listen-to-n-channels-dynamic-select-statement
	cases := []reflect.SelectCase{}

	currentIdx := 0
	appendCase := func(cs reflect.SelectCase) int {
		idx := currentIdx
		currentIdx++

		cases = append(cases, cs)

		return idx
	}

	appendRecvCase := func(ch interface{}) int {
		return appendCase(reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}

	interruptIdx := appendRecvCase(interruptChan)

	durationTimerIdx := -1
	if duration >= 0 {
		// we need to create timer's channels not in the cycle
		durationTimeout := time.Duration(float64(time.Second) * float64(duration))
		durationTimer := time.After(durationTimeout)
		durationTimerIdx = appendRecvCase(durationTimer)
	}

	rpsTickerIdx := -1
	defaultIdx := -1

	var rpsTicker *time.Ticker
	defer func() {
		if rpsTicker != nil {
			rpsTicker.Stop()
		}
	}()

	if rps >= 0 {
		rpsTimeout := time.Duration(float64(time.Second) / float64(rps))
		// we need ticker, not timer, to tick multiple times and stop after the cycle
		// (another way - just create time.After(rpsTimeout) just in the select-case)
		//rpsTimer := time.After(rpsTimeout)
		rpsTicker = time.NewTicker(rpsTimeout)
		rpsTickerIdx = appendRecvCase(rpsTicker.C)
	} else {
		defaultIdx = appendCase(reflect.SelectCase{
			Dir: reflect.SelectDefault,
		})
	}

	now := time.Now()
forCycle:
	for {
		// select {
		// case <-interruptChan:
		// 	break forCycle
		// case <-durationTimer:
		// 	break forCycle
		// case <-rpsTicker.C:
		// 	jc.StartJob(jobFunc)
		// default:
		// 	jc.StartJob(jobFunc)
		// }
		chosen, _, _ := reflect.Select(cases)
		switch chosen {
		case interruptIdx, durationTimerIdx:
			break forCycle
		case rpsTickerIdx, defaultIdx:
			jc.StartJob(jobFunc)
		}
	}
	measureTime := func() float64 {
		return time.Now().Sub(now).Seconds()
	}
	log.Infof("Stopped spawning jobs after %.2f seconds", measureTime())

	jc.Wait()
	return measureTime()
}

// :COPY_N_PASTE: Client
type Client struct {
	//Tags map[string]string
	//context *context

	url        string
	projectID  string
	authHeader string
}
