package stress

import (
	"encoding/json"
	"net/http"
	"regexp"
	"runtime"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("stress")

type HandlerFunc func(http.ResponseWriter, *http.Request, Match)

type Route struct {
	pattern string
	handler HandlerFunc
	pat     *regexp.Regexp
}

func NewRoute(pattern string, handler HandlerFunc) *Route {
	pat := regexp.MustCompile(pattern)
	return &Route{
		pattern: pattern,
		handler: handler,
		pat:     pat,
	}
}

func ServeJSON(w http.ResponseWriter, code int, jsonBytes []byte) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(jsonBytes)
}

func HttpError(w http.ResponseWriter, code int, msg ...string) {
	var o struct {
		Err string `json:"error,omitempty"`
	}
	if len(msg) != 0 {
		o.Err = msg[0]
	}

	b, _ := json.Marshal(&o)

	ServeJSON(w, code, b)
}

type HttpException struct {
	Code     int
	Messages []string
}

func RaiseHttpError(code int, msg ...string) {
	panic(HttpException{
		Code:     code,
		Messages: msg,
	})
}

type ServerHandler struct {
	routes []*Route
}

func (handler *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		// we should process exceptions ourselves, because:
		// - net/http just close TCP connection and log it via log.Printf() (not fine for aggregation system like Sentry)
		// - to handle RaiseHttpError() calls (github.com/gin-gonic/gin can't do it)

		// :TRICKY: we can catch anything but not panic(nil) ...
		// do not use panic(nil), bro
		if err := recover(); err != nil {
			if httpError, ok := err.(HttpException); ok {
				HttpError(w, httpError.Code, httpError.Messages...)
			} else {
				// copy-n-paste from http/net/server.go
				// for more clear file logging
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]

				log.Errorf("http panic: %s\n%s", err, buf)
				HttpError(w, http.StatusInternalServerError, "Something went wrong, bro")
			}
		}
	}()
	//fmt.Println(r.Method)
	//fmt.Println(r.URL)
	//fmt.Println(r.URL.Query())
	//fmt.Println(r.Header)
	//fmt.Println()

	path := r.URL.Path

	var route *Route
	var m []int
	for _, r := range handler.routes {
		m = r.pat.FindStringSubmatchIndex(path)
		if m != nil {
			route = r
			break
		}
	}

	if route != nil {
		route.handler(w, r, Match{
			m,
			path,
			route,
		})
	} else {
		http.NotFound(w, r)
	}
}

func Match2Map(m []int, s string, pat *regexp.Regexp) map[string]string {
	var result map[string]string

	if m != nil {
		result = make(map[string]string)

		for i, name := range pat.SubexpNames() {
			idx := m[2*i]
			if idx >= 0 {
				result[name] = s[idx:m[2*i+1]]
			}
		}
	}
	return result
}

type Match struct {
	match []int
	path  string
	route *Route
}

func (match Match) GetArgumentsMap() map[string]string {
	return Match2Map(match.match, match.path, match.route.pat)
}

func (match Match) Arguments() []string {
	var result []string

	pairs := match.match
	path := match.path
	for i := range pairs {
		if 2*i < len(pairs) && pairs[2*i] >= 0 {
			res := path[pairs[2*i]:pairs[2*i+1]]
			result = append(result, res)
		} else {
			break
		}
	}
	return result
}

func ListenAndServe(addr string, routes []*Route) {
	err := http.ListenAndServe(addr, &ServerHandler{
		routes,
	})
	CheckError(err)
}

type Duration time.Duration

func (d Duration) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(d).String()), nil
}
