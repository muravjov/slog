package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/G-Core/slog/base"
	"github.com/G-Core/slog/sentry"
	"github.com/erikdubbelboer/gspt"
)

func redirectToFd(dst *os.File, mappedFd int) {
	err := syscall.Dup2(int(dst.Fd()), mappedFd)
	base.CheckFatal("syscall.Dup2(): %s", err)
}

func startLaunchee(argv []string) *os.Process {
	// launchee start
	rpipe, wpipe, err := os.Pipe()
	base.CheckFatal("os.Pipe(): %s", err)

	f := []*os.File{
		os.Stdin,  // (0) stdin
		os.Stdout, // (1) stdout
		wpipe,     // (2) stderr
	}

	attr := &os.ProcAttr{
		Files: f,
	}

	launcheeP, err := os.StartProcess(argv[0], argv, attr)
	base.CheckFatal("Can't start launchee: %s", err)
	wpipe.Close()

	// redirect stdin to launchee stderr
	redirectToFd(rpipe, 0)

	return launcheeP
}

func DoMain(argv []string) {
	launcheeArgv := getLauncheeArgv(argv)
	bin := launcheeArgv[0]
	cfgOption, found := findConfigOptionFromArgs(argv)
	if !found {
		cfgOption = findConfigOptionFromUsage(bin)
	}

	cmd := exec.Command(bin,
		"--config",
		cfgOption,
		"logconfig",
	)
	bytes, err := cmd.Output()
	cmdStr := strings.Join(cmd.Args, " ")
	if err != nil {
		log.Fatalf("Cannot get logconfig (%s): %s", cmdStr, err)
	}

	//fmt.Println(string(bytes))
	dct := map[string]interface{}{}
	err = json.Unmarshal(bytes, &dct)
	if err != nil {
		log.Fatalf("Cannot unmarchal logconfig (%s): %s", cmdStr, err)
	}

	// disable SIGTERM exit -
	// do not let systemd or so stop the watcher before event is not processed.
	// https://golang.org/pkg/os/signal/#example_Notify
	c := make(chan os.Signal, 1)
	// signal.Notify invokes atomic change in sig.wanted, so even late SIGTERM signal
	// will not go after sigsend to dieFromSignal() (the latter is the source of die);
	// instead, the signal will go to channel c
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for s := range c {
			log.Printf("Launcher ignored signal: %s", s)
		}
	}()

	// setup launcher logging
	getCheckString := func(key string) (res string) {
		obj, exists := dct[key]
		if !exists {
			log.Printf(`Key "%s" doesn't exists in logconfig`, key)
			return
		}
		val, ok := obj.(string)
		if !ok {
			log.Printf(`Key "%s" in logconfig is not a string`, key)
			return
		}

		return val
	}

	logPath := getCheckString("log")
	if logPath != "" {
		// redirect launcher' stderr to errFileName
		logFile := base.OpenLog(logPath)

		redirectToFd(logFile, 2)
	}

	dsn := getCheckString("sentry_dsn")
	if dsn != "" {
		sentry.MustSetDSN(dsn)
		// no go-logging dep
		//SetupSentryLogger()
		sentry.SetSEH(func(err error) {
			log.Print(err)
		})
	}

	launcheeP := startLaunchee(launcheeArgv)

	// change process name
	s := fmt.Sprintf("Launcher for pid: %d", launcheeP.Pid)
	//log.Print(s)
	gspt.SetProcTitle(s)

	base.ProcessStream(os.Stdin, launcheeP.Pid, launcheeArgv)

	ps, err := launcheeP.Wait()
	base.CheckFatal("Error while waiting for launchee: %s", err)

	exitCode := 0
	if !ps.Success() {
		log.Printf("Launchee exit is not 0: %s", ps)
		exitCode = 1
	}
	os.Exit(exitCode)
}
