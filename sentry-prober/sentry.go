package main

import (
	"fmt"
	"net/http"
	urlModule "net/url"
	"path"
	"runtime"
	"strings"

	"github.com/G-Core/slog/base"
	"github.com/G-Core/slog/sentry"
	raven "github.com/getsentry/raven-go"
)

func MakeClient(dsn string) *Client {
	client := &Client{}

	SetDSN := func(dsn string) error {
		if dsn == "" {
			return nil
		}

		uri, err := urlModule.Parse(dsn)
		if err != nil {
			return err
		}

		if uri.User == nil {
			return raven.ErrMissingUser
		}
		publicKey := uri.User.Username()
		secretKey, hasSecretKey := uri.User.Password()
		uri.User = nil

		if idx := strings.LastIndex(uri.Path, "/"); idx != -1 {
			client.projectID = uri.Path[idx+1:]
			uri.Path = uri.Path[:idx+1] + "api/" + client.projectID + "/store/"
		}
		if client.projectID == "" {
			return raven.ErrMissingProjectID
		}

		client.url = uri.String()

		if hasSecretKey {
			client.authHeader = fmt.Sprintf("Sentry sentry_version=4, sentry_key=%s, sentry_secret=%s", publicKey, secretKey)
		} else {
			client.authHeader = fmt.Sprintf("Sentry sentry_version=4, sentry_key=%s", publicKey)
		}

		return nil
	}

	err := SetDSN(dsn)
	base.CheckError(err)

	return client
}

type Event struct {
	isError  bool
	key      string
	isRandom bool
}

func PostSentryEvent(ev *Event, client *Client, rc *RequestContext) (eventID string, err error) {
	var args []interface{} = nil

	message := ev.key
	if ev.isRandom {
		arg := base.RandStringBytes(8)
		args = []interface{}{
			arg,
		}
		message = fmt.Sprintf(ev.key, arg)
	}

	calldepth := 1
	var captureTags map[string]string

	var packet *raven.Packet
	if ev.isError {
		// :COPY_N_PASTE: CaptureErrorAndWait()
		var level raven.Severity = raven.ERROR

		stacktrace := raven.NewStacktrace(calldepth, 3, nil)
		packet = sentry.Interface2Packet(message, stacktrace, level)
	} else {
		iObject := &raven.Message{
			Message: ev.key,
			Params:  args,
		}

		// :COPY_N_PASTE: CaptureMessageAndWait()
		packet = sentry.Interface2Packet(message, iObject, raven.WARNING)

		var fn string
		pc, pathname, line, ok := runtime.Caller(calldepth)
		if ok {
			fn = path.Base(pathname)
		}
		_ = pc

		if ok {
			extra := packet.Extra
			extra["filename"] = fn
			extra["lineno"] = line
			extra["pathname"] = pathname
		}
	}

	// :COPY_N_PASTE: func (client *Client) Capture()
	packet.AddTags(captureTags)
	// :TODO: SetUserContext(), SetHttpContext(), SetTagsContext()
	//packet.AddTags(client.Tags)
	//packet.AddTags(client.context.tags)

	err = packet.Init(client.projectID) // установка Timestamp и прочего
	if err != nil {
		return
	}

	url, authHeader := client.url, client.authHeader
	// :COPY_N_PASTE: func (t *HTTPTransport) Send(url, authHeader string, packet *raven.Packet) error
	if url == "" {
		return
	}

	packetJSON, err := packet.JSON()
	if err != nil {
		err = fmt.Errorf("error marshaling packet %+v to JSON: %v", packet, err)
		return
	}
	//fmt.Println(string(packetJSON))

	headers := map[string]string{
		"X-Sentry-Auth": authHeader,
		"User-Agent":    userAgent,
		"Content-Type":  "application/json",
	}

	body, contentType, err := serializedPacket(packetJSON)
	if err != nil {
		err = fmt.Errorf("error serializing packet: %v", err)
		return
	}
	headers["Content-Type"] = contentType

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		err = fmt.Errorf("can't create new request: %v", err)
		return
	}

	for key, val := range headers {
		req.Header.Add(key, val)
	}

	// res, err := httpClient.Do(req)
	// if err != nil {
	// 	return
	// }
	// io.Copy(ioutil.Discard, res.Body)
	// res.Body.Close()
	// if res.StatusCode != 200 {
	// 	err = fmt.Errorf("raven: got http status %d", res.StatusCode)
	// 	return
	// }
	ExecuteRequest(req, rc)

	return packet.EventID, nil
}
