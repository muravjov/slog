package stress

import (
	"encoding/json"
	"net/http"
	"time"
)

type RouteHelp struct {
	Pattern string
	Help    string
}

type CheckData struct {
	Version       string    `json:"version"`
	Start         time.Time `json:"start"`
	UptimeSeconds float64   `json:"uptime_seconds"`
}

func MakeCheckData(version string) CheckData {
	now := time.Now()
	return CheckData{
		Version:       version,
		Start:         now,
		UptimeSeconds: 0,
	}
}

type Routes []*Route

type ServeData struct {
	routeList []RouteHelp
	checkData CheckData
}

func NewServeData(version string) *ServeData {
	return &ServeData{
		checkData: MakeCheckData(version),
	}
}

type RouteFunc func(pattern string, handler HandlerFunc, optHelp ...string) *Route

func GenerateMakeRoute(serveData *ServeData) RouteFunc {

	return func(pattern string, handler HandlerFunc, optHelp ...string) *Route {
		help := ""
		if len(optHelp) > 0 {
			help = optHelp[0]
		}

		serveData.routeList = append(serveData.routeList, RouteHelp{
			Pattern: pattern,
			Help:    help,
		})
		return NewRoute(pattern, handler)
	}
}

func MarshalIndent(v interface{}) []byte {
	dat, err := json.MarshalIndent(v, "", "\t")
	CheckError(err)

	return dat
}

func Serve(addr string, appRoutes Routes, serveData *ServeData) {
	makeRoute := GenerateMakeRoute(serveData)

	commonRoutes := Routes{
		makeRoute("^/check$", func(w http.ResponseWriter, r *http.Request, match Match) {
			dat := serveData.checkData
			dat.UptimeSeconds = time.Now().Sub(dat.Start).Seconds()
			ServeJSON(w, http.StatusOK, MarshalIndent(dat))
		}, "Service version and uptime"),

		makeRoute("^/aux/help$", func(w http.ResponseWriter, r *http.Request, match Match) {
			b := MarshalIndent(serveData.routeList)
			ServeJSON(w, http.StatusOK, b)
		}, "Show help about all routes/endpoints"),
	}

	var routes Routes
	for _, r := range []Routes{
		appRoutes,
		commonRoutes,
	} {
		routes = append(routes, r...)
	}

	ListenAndServe(addr, routes)
}
