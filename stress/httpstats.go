package stress

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	urlModule "net/url"
	"os"
	"reflect"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/G-Core/slog/base"
)

type Statuses map[string]int64

type StatsType struct {
	Statuses Statuses

	// contains the list of all done requests
	// 10 seconds of heavy stress doesn't eat much memory:
	// 10 seconds * 6000 prs = 60000 float64 = 60000 * 8 = 480000 < 0.5 Megabyte
	Times []float64
}

type RequestContext struct {
	Client *http.Client
	Stats  *StatsType

	IncrStat func(msg string, dur float64)

	WaitStatsReady func()
}

type RequestContextOptions struct {
	KeepAlive     bool
	StressTimeout float64
}

func NewRequestContext() *RequestContext {
	rco := &RequestContextOptions{
		KeepAlive: false,
	}
	return NewRequestContextEx(rco)
}

func NewRequestContextEx(rco *RequestContextOptions) *RequestContext {
	rc := &RequestContext{}

	var rt http.RoundTripper
	if !rco.KeepAlive {
		transport := &http.Transport{}
		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		base.Assert(ok)

		// :TRICKY: make altered variant of http.DefaultTransport - no way to make simpler :(
		// "assigment copies lock value (sync.Mutex)"
		//*transport = *defaultTransport
		toElem := func(i interface{}) reflect.Value {
			return reflect.ValueOf(i).Elem()
		}
		tValue, dtValue := toElem(transport), toElem(defaultTransport)
		// list of attrs see at http.DefaultTransport initialization
		for _, name := range []string{
			"Proxy",
			"DialContext",
			"MaxIdleConns",
			"IdleConnTimeout",
			"TLSHandshakeTimeout",
			"ExpectContinueTimeout",
		} {
			tValue.FieldByName(name).Set(dtValue.FieldByName(name))
		}
		transport.DisableKeepAlives = true

		rt = transport
	}

	client := &http.Client{Transport: rt}
	rc.Client = client
	client.Timeout = time.Duration(float64(time.Second) * rco.StressTimeout)

	// for the sake of simplicity, let's choose sync.Map
	// https://medium.com/@deckarep/the-new-kid-in-town-gos-sync-map-de24a6bf7c2c
	//var stats sync.Map
	//
	// No, unnecessarily difficult for now,- go for channels
	statuses := make(map[string]int64)
	stats := &StatsType{
		statuses,
		nil,
	}
	rc.Stats = stats

	type requestStat struct {
		msg      string
		duration float64 // seconds
	}

	statChan := make(chan requestStat, 1000)
	statDone := NewEvent()
	go func() {
		for rs := range statChan {
			msg := rs.msg
			counter, ok := statuses[msg]
			if ok {
				counter = counter + 1
			} else {
				counter = 1
			}
			statuses[msg] = counter

			stats.Times = append(stats.Times, rs.duration)
		}
		statDone.Done()
	}()

	rc.IncrStat = func(msg string, dur float64) {
		statChan <- requestStat{msg, dur}
	}

	rc.WaitStatsReady = func() {
		close(statChan)
		statDone.Wait()
	}

	return rc
}

// :TRICKY: see http.errServerClosedIdle - upexported
var errServerClosedIdle error

func IsServerClosedIdle(err error) bool {
	if errServerClosedIdle != nil {
		return err == errServerClosedIdle
	}

	if err.Error() == "http: server closed idle connection" {
		errServerClosedIdle = err
		return true
	}

	return false
}

func ExecuteRequest(req *http.Request, rc *RequestContext) {
	// https://blog.golang.org/http-tracing
	gotConn, gotFirstResponseByte := false, false
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			gotConn = true
		},
		GotFirstResponseByte: func() {
			gotFirstResponseByte = true
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	now := time.Now()
	appendStat := func(msg string) {
		dur := measureTime(now)
		rc.IncrStat(msg, dur)
	}

	// :TRICKY: main error checking occurs in transport.go:persistConn.readLoop()
	resp, err := rc.Client.Do(req)
	if err != nil {
		var hint string
		// client.Do() wraps real error with "method: url: <real error>",
		// it is inconveniently
		switch errTyped := err.(type) {
		case *urlModule.Error:
			err = errTyped.Err
			if err == io.EOF {
				// so weird looking "EOF"
				hint = fmt.Sprintf("GotConn=%t, GotFirstResponseByte=%t", gotConn, gotFirstResponseByte)
			} else if IsServerClosedIdle(err) {
				hint = "client stays keepalive after request, but server closes the connection"
			} else if errTyped.Timeout() {
				// really that is err.(net.Error) and unexported httpError
				hint = "non-zero timeout hit"
			}
		}

		msg := err.Error()
		if hint != "" {
			msg = fmt.Sprintf("%s (%s)", msg, hint)
		}
		appendStat(msg)
		return
	}
	defer resp.Body.Close()

	//fmt.Println(resp.StatusCode)
	//bdat, err := ioutil.ReadAll(resp.Body)
	//base.CheckError(err)
	//fmt.Println(string(bdat))
	io.Copy(ioutil.Discard, resp.Body)

	switch dig := resp.StatusCode / 100; dig {
	case 2, 4, 5:
		appendStat(fmt.Sprintf("%dxx", dig))
	default:
		appendStat(resp.Status)
	}
}

func ExecuteGet(url string, rc *RequestContext) {
	var body io.Reader = nil
	req, err := http.NewRequest("GET", url, body)
	base.CheckError(err)

	//req.Header.Add("If-None-Match", `W/"wyzzy"`)
	ExecuteRequest(req, rc)
}

type StatElement struct {
	Key     string
	Counter int64
}

type StatList []StatElement

func (p StatList) Len() int { return len(p) }

// reverse order
func (p StatList) Less(i, j int) bool { return p[j].Counter < p[i].Counter }
func (p StatList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func PrintReport(stats *StatsType, stressTimes StressTimes, aggregationCount int) {
	statuses := stats.Statuses

	var lst StatList
	var totals int64 = 0
	successTotals := statuses["2xx"]
	for k, v := range statuses {
		lst = append(lst, StatElement{k, v})
		totals += v
	}

	sort.Sort(lst)

	if len(lst) > aggregationCount {
		tail := lst[aggregationCount-1:]

		lst = lst[:aggregationCount-1]

		var tailCounter int64 = 0
		for _, se := range tail {
			tailCounter += se.Counter
		}

		lst = append(lst, StatElement{
			"Other Stats",
			tailCounter,
		})

		// again, to make "Other Stats" in the right order
		sort.Sort(lst)
	}

	getRatio := func(counter int64, totals float64) string {
		percent := "n/a"
		if totals > 0 {
			percent = fmt.Sprintf("%.2f", float64(counter)/float64(totals))
		}
		return percent
	}

	getPercent := func(counter int64, totals float64) string {
		return getRatio(counter*100, totals)
	}

	// Print
	fmt.Print(`
=======
Report

`)
	fmt.Println("Counter Table:")
	fmt.Printf("%%\t%s\t%s\n", "Counter", "Name")
	for _, se := range lst {
		fmt.Printf("%s\t%d\t\"%s\"\n", getPercent(se.Counter, float64(totals)), se.Counter, se.Key)
	}

	fmt.Println("\nTotals:")
	const padding = 3
	var twFlags uint = 0 // tabwriter.Debug
	w := tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', twFlags)
	writeColumn := func(format string, a ...interface{}) {
		fmt.Fprintln(w, fmt.Sprintf(format, a...))
	}

	timeElapsed := stressTimes.ElapsedTime
	writeColumn("Number of Requests:\t%d", totals)
	writeColumn("Time Elapsed:\t%.2f seconds", timeElapsed)
	writeColumn("Requests per Second:\t%s", getRatio(totals, timeElapsed))
	writeColumn("Jobs Spawning per Second:\t%s", getRatio(totals, stressTimes.SpawningJobsTime))
	writeColumn("Success percent (HTTP 2xx):\t%s", getPercent(successTotals, float64(totals)))

	w.Write([]byte("\n"))

	times := stats.Times
	sort.Float64s(times)

	writeTime := func(title string, fn func() float64) {
		value := "n/a"
		if ln := len(times); ln > 0 {
			value = fmt.Sprintf("%f", fn())
		}
		writeColumn("%s, seconds:\t%s", title, value)
	}

	writeTime("Worst request duration", func() float64 {
		return times[len(times)-1]
	})
	writeTime("Mean  request duration", func() float64 {
		var sum float64
		for _, d := range times {
			sum += d
		}

		return sum / float64(len(times))
	})
	writeTime("95ptl request duration", func() float64 {
		idx := int(float64(len(times)) * 0.95)
		return times[idx]
	})

	w.Flush()
}
