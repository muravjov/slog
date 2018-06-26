package main

import (
	"compress/zlib"
	"fmt"
	"runtime/pprof"
	"sync"

	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/G-Core/slog"
	"github.com/G-Core/slog/base"
	"github.com/G-Core/slog/sentry"
	slogV2 "github.com/G-Core/slog/v2"
	"github.com/getsentry/raven-go"
	flag "github.com/spf13/pflag"

	"github.com/G-Core/slog/watcher"
	"github.com/kballard/go-shellquote"
	"github.com/pkg/errors"
)

var userAgent = "slog/1.0"

type HTTPTransport struct {
	Mode string
	*http.Client
}

func Shell2Cmd(cmdStr string) (*exec.Cmd, error) {
	command, err := shellquote.Split(cmdStr)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(command[0], command[1:]...)
	return cmd, nil
}

func GotErrorEx(isError bool, format string, args ...interface{}) bool {
	if isError {
		log.Errorf(format, args...)
		return true
	}
	return false
}

func StartProcess(cmdStr string) *exec.Cmd {
	cmd, err := Shell2Cmd(cmdStr)
	if GotErrorEx(err != nil, "Bad command string \"%s\": %s", err, cmdStr) {
		return nil
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()

	if GotErrorEx(err != nil, "Can't start command \"%s\": %s", err, cmdStr) {
		return nil
	}
	return cmd
}

func WaitProcess(cmd *exec.Cmd, cmdStr string) error {
	err := cmd.Wait()
	if err != nil {
		err = errors.Errorf("Command ended badly \"%s\": %s", err, cmdStr)
	}

	return err
}

func SpawnProcess(cmdStr string) {
	cmd := StartProcess(cmdStr)

	// - you have to wait until the end of the child or zombie process would be
	// - one more goroutine is not that big for waiting it, it is ok for Go to behave so (goroutine << dead process)
	// - you may notify SIGCHLD but it is overkill in 99.9999% situation (only valid IMHO if you are PID 1 process)
	// - you may create special goroutine and delegate waiting a list of spawned processes, but it is not that easy and
	//   doesn't work with exec.Command (you need your own realisation)
	go func() {
		err := WaitProcess(cmd, cmdStr)
		if err != nil {
			log.Error(err)
		}
	}()
}

func (t *HTTPTransport) Send(url, authHeader string, packet *raven.Packet) error {
	if url == "" {
		return nil
	}

	packetJSON, err := packet.JSON()
	if err != nil {
		return fmt.Errorf("error marshaling packet %+v to JSON: %v", packet, err)
	}

	headers := map[string]string{
		"X-Sentry-Auth": authHeader,
		"User-Agent":    userAgent,
		"Content-Type":  "application/json",
	}

	switch t.Mode {
	case "slog":
		body, contentType, err := serializedPacket(packetJSON)
		if err != nil {
			return fmt.Errorf("error serializing packet: %v", err)
		}
		headers["Content-Type"] = contentType

		req, err := http.NewRequest("POST", url, body)
		if err != nil {
			return fmt.Errorf("can't create new request: %v", err)
		}

		for key, val := range headers {
			req.Header.Add(key, val)
		}
		res, err := t.Do(req)
		if err != nil {
			return err
		}
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			return fmt.Errorf("raven: got http status %d", res.StatusCode)
		}
	case "curl-print", "curl-execute":
		// :TRICKY: curl is not able to compress POST data; we also do not want to make commands with binary data
		// so no way to compress big events
		hList := []string{}
		for key, val := range headers {
			hList = append(hList, "-H", shellquote.Join(fmt.Sprintf("%s: %s", key, val)))
		}

		cmd := fmt.Sprintf("curl -X POST %s %s --data-binary %s", strings.Join(hList, " "), shellquote.Join(url), shellquote.Join(string(packetJSON)))
		fmt.Println(cmd)

		if t.Mode == "curl-execute" {
			cmdObj := StartProcess(cmd)
			err := WaitProcess(cmdObj, cmd)
			if err != nil {
				return err
			}
		}
	default:
		base.Assert(false)
	}

	return nil
}

var compressionLevel int = zlib.BestSpeed

//var compressionLevel int = zlib.BestCompression

var deflateFree = sync.Pool{
	New: func() interface{} {
		deflate, err := zlib.NewWriterLevel(nil, compressionLevel)
		base.CheckError(err)
		return deflate
	},
}

func serializedPacket(packetJSON []byte) (io.Reader, string, error) {
	// Only deflate/base64 the packet if it is bigger than 1KB, as there is
	// overhead.
	if len(packetJSON) > 1000 {
		buf := &bytes.Buffer{}
		b64 := base64.NewEncoder(base64.StdEncoding, buf)

		//deflate, _ := zlib.NewWriterLevel(b64, compressionLevel)
		//base.CheckError(err)
		deflate, ok := deflateFree.Get().(*zlib.Writer)
		base.Assert(ok)
		deflate.Reset(b64)
		defer deflateFree.Put(deflate)

		deflate.Write(packetJSON)
		deflate.Close()
		b64.Close()
		return buf, "application/octet-stream", nil
	}
	return bytes.NewReader(packetJSON), "application/json", nil
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "save CPU profile to file")

	isWarning := flag.BoolP("warning", "", false, "warning (without stacktrace) vs error")
	pMessage := flag.StringP("message", "", "sentry-prober", "message to send")
	isRandom := flag.BoolP("random", "", false, "add random string to message")
	isWatcher := flag.BoolP("watcher", "", false, "send message via watcher")
	transport := flag.StringP("transport", "", "default",
		"transport to use (not for --watcher): default|raven-go|slog|curl-print|curl-execute")

	isStress := flag.Bool("stress", false, "run stress test")
	stressRPS := flag.Float64("stress-rps", 5., "stress: request per second; < 0 means requesting without time throttling")
	stressDuration := flag.Float64("stress-duration", -1., "stress duration; < 0 means stress to be stopped with Ctrl+C")
	stressReqNumber := flag.Int("stress-request-number", -1,
		"N >=0 means direct stress mode \"for range N {StartJob()}\"")
	selfStress := flag.Bool("stress-self", false, "start dummy http server at dsn")
	keepaliveStress := flag.Bool("stress-keepalive", false, "reuse TCP connections between different HTTP requests")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("Error creating cpu profile: %s\n", err)
		}

		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// :TODO: sentry-prober --help to show required argument: DSN
	dsn := flag.Arg(0)
	if dsn == "" {
		log.Fatalf("DSN argument required")
	}

	if *isStress {
		if *selfStress {
			go ServeDummyHTTP(dsn)
		}

		//fmt.Println(*stressRPS, *stressDuration)
		rco := &RequestContextOptions{
			KeepAlive: *keepaliveStress,
		}
		rc := NewRequestContextEx(rco)

		event := &Event{
			isError:  !*isWarning,
			key:      *pMessage,
			isRandom: *isRandom,
		}

		client := MakeClient(dsn)

		jobFunc := func() {
			eventID, err := PostSentryEvent(event, client, rc)
			base.CheckError(err)
			_ = eventID
		}
		stressTimes := MakeStress(jobFunc, *stressRPS, *stressDuration, *stressReqNumber)

		rc.WaitStatsReady()
		PrintReport(rc.Stats, stressTimes, 10)

		return
	}

	slogV2.MustSetDSNAndHandler(dsn)

	ht, ok := raven.DefaultClient.Transport.(*raven.HTTPTransport)
	base.Assert(ok)

	switch *transport {
	case "default", "raven-go":
	case "slog", "curl-print", "curl-execute":
		t := &HTTPTransport{
			Mode: *transport,
			// transfer settings to new transport
			Client: ht.Client,
		}

		raven.DefaultClient.Transport = t
	default:
		log.Fatalf("Unknown transport: \"%s\"", *transport)
	}

	message := *pMessage

	key := message
	var args []interface{} = nil
	if *isRandom {
		arg := base.RandStringBytes(8)
		args = []interface{}{
			arg,
		}
		message = fmt.Sprintf(key, arg)
	}

	if *isWatcher {
		watcher.StartWatcher(dsn, "")

		fmt.Printf(`Do exception to invoke watcher to send post-mortem
`)

		slog.ForceException()
	} else {
		fmt.Printf(`Sending "%s"...
`, message)

		var eventID string
		if *isWarning {
			eventID = sentry.CaptureMessageAndWait(message, nil, 0, &raven.Message{
				Message: key,
				Params:  args,
			})
		} else {
			eventID = sentry.CaptureErrorAndWait(message, nil, 0, raven.ERROR)
		}

		fmt.Printf("Event ID: %s\n", eventID)
	}
}
