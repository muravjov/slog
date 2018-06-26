package main

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

func ServeDummyHTTP(dsn string) {
	uri, err := url.Parse(dsn)
	CheckError(err)

	serveData := NewServeData("n/a")
	makeRoute := GenerateMakeRoute(serveData)

	routes := Routes{
		// url(r'^api/(?P<project_id>[\w_-]+)/store/$', api.StoreView.as_view(), name='sentry-api-store')
		makeRoute(`^/api/(?P<project_id>[\w_-]+)/store/$`, func(w http.ResponseWriter, r *http.Request, match Match) {
			s := match.GetArgumentsMap()["project_id"]
			_ = s
			//fmt.Println(s)

			// :TODO: go-raven sends events as "application/octet-stream" and evidently gzip-es them
			// var dat StrDict
			// err := DecodeCheckJSONBody(w, r, &dat)
			// if err != nil {
			// 	return
			// }
			//spew.Dump(dat)

			b := MarshalIndent(map[string]string{})
			ServeJSON(w, http.StatusCreated, b)
		}, "sentry-api-store emulation"),
	}

	//fmt.Println(uri.Host)
	Serve(uri.Host, routes, serveData)
	CheckError(err)
}
