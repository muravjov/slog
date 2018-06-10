package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandLine(t *testing.T) {
	val, found := findConfigOptionFromArgs([]string{
		"launcher",
		"./bin/service",
		"ggg",
		"--config1", "./agent1.toml",
		"--config", "./service.toml",
		"--config", "./service.toml",
	})
	require.True(t, found)
	require.Equal(t, "./service.toml", val)

	val, found = findConfigOptionFromArgs([]string{
		"launcher",
		"./bin/service",
		"ggg",
		"--config1", "./service.toml",
	})
	require.False(t, found)
}

func TestFromHelp(t *testing.T) {
	output := `Usage of ./bin/service:
    --config string   service config file (default "/etc/service/service.toml")
    --version         Show version and exit
pflag: help requested`
	fname := findConfigFromOutput(output)
	_ = output
	require.Equal(t, "/etc/service/service.toml", fname)

	if false {
		bin := "./bin/agent"

		fmt.Println(findConfigOptionFromUsage(bin))
	}
}

func TestLaunch(t *testing.T) {
	t.SkipNow()

	argv := []string{
		"launcher",
		"bin/master", "--config",
		"./master.toml",
		//"tmp/master.toml",
		"server",
	}

	DoMain(argv)
}
