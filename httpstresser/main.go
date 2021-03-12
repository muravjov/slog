package main

import (
	"log"
	"net/http"

	"github.com/muravjov/slog/stress"
	flag "github.com/spf13/pflag"
)

func main() {
	// :TODO:
	//cpuprofile := flag.String("cpuprofile", "", "save CPU profile to file")

	stressRPS := flag.Float64("rps", 5., "stress: request per second; < 0 means requesting without time throttling")
	stressDuration := flag.Float64("duration", -1., "stress duration; < 0 means stress to be stopped with Ctrl+C")
	stressReqNumber := flag.Int("request-number", -1,
		"N >=0 means direct stress mode \"for range N {StartJob()}\"")
	selfStress := flag.Bool("self", false, "start dummy http server at url")
	keepaliveStress := flag.Bool("keepalive", false, "reuse TCP connections between different HTTP requests")
	stressTimeout := flag.Float64("timeout", 5., "stress timeout; = 0 means no request timeout")
	tls := flag.String("tls", "", "Custom tls config file: toml format, fields ca, cert, key, skip_verify, server_name")
	selfTls := flag.String("self-tls", "", "Custom tls config file, for --self server")

	flag.Parse()

	// :TODO: sentry-prober --help to show required argument: URI
	uri := flag.Arg(0)
	if uri == "" {
		log.Fatalf("URI argument required")
	}

	if *selfStress {
		route := stress.NewRoute(`.*`, func(w http.ResponseWriter, r *http.Request, match stress.Match) {
			b := stress.MarshalIndent(map[string]string{})
			stress.ServeJSON(w, http.StatusOK, b)
		})

		go stress.ServeDummyHTTP(uri, route, *selfTls)
	}

	//fmt.Println(*stressRPS, *stressDuration)
	rco := &stress.RequestContextOptions{
		KeepAlive:     *keepaliveStress,
		StressTimeout: *stressTimeout,
		TLSConfigFile: *tls,
	}
	rc := stress.NewRequestContextEx(rco)

	jobFunc := func() {
		stress.ExecuteGet(uri, rc)
	}
	stressTimes := stress.MakeStress(jobFunc, *stressRPS, *stressDuration, *stressReqNumber)

	rc.WaitStatsReady()
	stress.PrintReport(rc.Stats, stressTimes, 10)
}
