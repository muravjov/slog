package stress

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/muravjov/slog/base"
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

func ServeDummyHTTP(targetUrl string, route *Route, serverTLSConfigFname string) {
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

	var tlsConfig *tls.Config
	switch uri.Scheme {
	case "http":
	case "https":
		if serverTLSConfigFname != "" {
			tlsConfig = NewServerTLSConfig(serverTLSConfigFname)
		}
		// :TRICKY: in case of https we anyway make non nil *tls.Config
		// to force https serving
		if tlsConfig == nil {
			tlsConfig = &tls.Config{}
		}
	default:
		log.Fatalf("unknown url scheme: %s", uri.Scheme)
	}

	//fmt.Println(uri.Host)
	Serve(uri.Host, routes, serveData, tlsConfig)
	CheckError(err)
}
