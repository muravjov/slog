package main

import (
	"net/url"

	"github.com/G-Core/slog/base"
)

var CheckError = base.CheckError

func ServeDummyHTTP(dsn string) {
	uri, err := url.Parse(dsn)
	CheckError(err)

	serveData := NewServeData("n/a")
	makeRoute := GenerateMakeRoute(serveData)

	routes := Routes{}
	_ = makeRoute

	//fmt.Println(uri.Host)
	Serve(uri.Host, routes, serveData)
	CheckError(err)
}
