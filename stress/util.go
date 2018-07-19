package stress

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/G-Core/slog/base"
)

var CheckError = base.CheckError

func DecodeCheckJSONBody(w http.ResponseWriter, r *http.Request, v interface{}) (err error) {
	err = json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		HttpError(w, http.StatusBadRequest, fmt.Sprintf("Bad json, bro: %s", err))
		return
	}

	return nil
}

type StrDict map[string]interface{}

func ServeDummyHTTP(targetUrl string, route *Route) {
	uri, err := url.Parse(targetUrl)
	CheckError(err)

	serveData := NewServeData("n/a")
	//makeRoute := GenerateMakeRoute(serveData)

	serveData.routeList = append(serveData.routeList, RouteHelp{
		Pattern: route.pattern,
		Help:    "stress handler",
	})

	routes := Routes{
		route,
	}

	//fmt.Println(uri.Host)
	Serve(uri.Host, routes, serveData)
	CheckError(err)
}
