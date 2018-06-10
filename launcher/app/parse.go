package app

import (
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

// like in github.com/spf13/pflag/string.go
type stringValue string

func newStringValue() *stringValue {
	return new(stringValue)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}
func (s *stringValue) Type() string {
	return "string"
}

func (s *stringValue) String() string { return string(*s) }

func getLauncheeArgv(launcerArgv []string) []string {
	if len(launcerArgv) < 2 {
		log.Fatal("Launcher needs an argument to execute")
	}

	// so, if launchee wants to use -- too then launcher command should include
	// -- twice, e.g.
	// $ launcher --option ... -- launchee --anotheroption ... -- arg
	argv := launcerArgv[1:]
	for idx, s := range argv {
		if s == "--" {
			argv = argv[idx+1:]
			break
		}
	}
	return argv
}

func findConfigOptionFromArgs(argv []string) (string, bool) {
	argv = getLauncheeArgv(argv)[1:]

	var flagSet = flag.NewFlagSet("", flag.ContinueOnError)
	// not to exit in case of unknown options
	flagSet.ParseErrorsWhitelist.UnknownFlags = true

	configFlag := &flag.Flag{
		Name:  "config",
		Value: newStringValue(),
	}
	flagSet.AddFlag(configFlag)

	err := flagSet.Parse(argv)
	if err != nil {
		log.Fatalf("Can't parse arguments, %s: %v", err, argv)
	}

	return configFlag.Value.String(), configFlag.Changed
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

//var configPat = regexp.MustCompile(`(?m) --config .* \(default (?P<default>".+")\)$`)
var configPat = regexp.MustCompile(`(?m) --config .*\(default (?P<default>".+")\)`)

func findConfigOptionFromUsage(bin string) string {
	argv := []string{"--help"}

	cmd := exec.Command(bin, argv...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		// github.com/spf13/pflag gives exit code 2, but github.com/spf13/cobra - 0
		_, ok := err.(*exec.ExitError)
		if !ok {
			log.Fatalf("Cannot start program for help usage (%s): %s", strings.Join(cmd.Args, " "), err)
		}
	}

	out := string(bytes)

	return findConfigFromOutput(out)
}

func findConfigFromOutput(out string) string {
	m := configPat.FindStringSubmatchIndex(out)
	if m == nil {
		log.Fatalf(`Cannot find --config' default from program usage:
%s`, out)
	}

	val := Match2Map(m, out, configPat)["default"]
	configFname, err := strconv.Unquote(val)
	if err != nil {
		log.Fatalf("Cannot strconv.Unquote() configFname: %s", val)
	}

	return configFname
}
