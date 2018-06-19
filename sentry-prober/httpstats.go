package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	urlModule "net/url"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/G-Core/slog/base"
)

type StatsType map[string]int64

type RequestContext struct {
	Client *http.Client
	Stats  StatsType

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

func ExecuteRequest(req *http.Request, rc *RequestContext) {
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
	io.Copy(ioutil.Discard, resp.Body)

	switch dig := resp.StatusCode / 100; dig {
	case 2, 4, 5:
		rc.IncrStat(fmt.Sprintf("%dxx", dig))
	default:
		rc.IncrStat(resp.Status)
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

func PrintReport(stats StatsType, timeElapsed float64, aggregationCount int) {
	var lst StatList
	var totals int64 = 0
	successTotals := stats["2xx"]
	for k, v := range stats {
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

	writeColumn("Number of Requests:\t%d", totals)
	writeColumn("Time Elapsed:\t%.2f seconds", timeElapsed)
	writeColumn("Requests per Second:\t%s", getRatio(totals, timeElapsed))
	writeColumn("Success percent (HTTP 2xx):\t%s", getPercent(successTotals, float64(totals)))
	w.Flush()
}
