package core

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tritondatacenter/containerpilot/subcommands"
	"github.com/tritondatacenter/containerpilot/version"
)

// MultiFlag provides a custom CLI flag that stores its unique values into a
// simple map.
type MultiFlag struct {
	Values map[string]string
}

// String satisfies the flag.Value interface by joining together the flag values
// map into a single String.
func (f MultiFlag) String() string {
	return fmt.Sprintf("%v", f.Values)
}

// Set satisfies the flag.Value interface by creating a map of all unique CLI
// flag values.
func (f *MultiFlag) Set(value string) error {
	if f.Len() == 0 {
		f.Values = make(map[string]string, 1)
	}
	pair := strings.SplitN(value, "=", 2)
	if len(pair) < 2 {
		return fmt.Errorf("flag value '%v' was not in the format 'key=val'", value)
	}
	f.Values[pair[0]] = pair[1]
	return nil
}

// Len is the length of the slice of values for this MultiFlag.
func (f MultiFlag) Len() int {
	return len(f.Values)
}

// GetArgs parses the command line flags and returns the subcommand
// we need and its parameters (if any)
func GetArgs() (subcommands.Handler, subcommands.Params) {

	var versionFlag bool
	var templateFlag bool
	var reloadFlag bool
	var pingFlag bool

	var configPath string
	var renderFlag string
	var maintFlag string

	var putMetricFlags MultiFlag
	var putEnvFlags MultiFlag

	if !flag.Parsed() {
		flag.BoolVar(&versionFlag, "version", false,
			"Show version identifier and quit.")

		flag.BoolVar(&templateFlag, "template", false,
			"Render template and quit.")

		flag.BoolVar(&reloadFlag, "reload", false,
			"Reload a ContainerPilot process through its control socket.")

		flag.StringVar(&configPath, "config", "",
			"File path to JSON5 configuration file. Defaults to CONTAINERPILOT env var.")

		flag.StringVar(&renderFlag, "out", "",
			`File path where to save rendered config file when '-template' is used.
	Defaults to stdout ('-').`)

		flag.StringVar(&maintFlag, "maintenance", "",
			`Toggle maintenance mode for a ContainerPilot process through its control socket.
	Options: '-maintenance enable' or '-maintenance disable'`)

		flag.Var(&putMetricFlags, "putmetric",
			`Update metrics of a ContainerPilot process through its control socket.
	Pass metrics in the format: 'key=value'`)

		flag.Var(&putEnvFlags, "putenv",
			`Update environ of a ContainerPilot process through its control socket.
	Pass environment in the format: 'key=value'`)

		flag.BoolVar(&pingFlag, "ping", false,
			"Check that the ContainerPilot control socket is up.")

		flag.Parse()
	}

	if versionFlag {
		return subcommands.VersionHandler, subcommands.Params{
			Version: version.Version,
			GitHash: version.GitHash,
		}
	}
	if configPath == "" {
		configPath = os.Getenv("CONTAINERPILOT")
	}
	if templateFlag {
		return subcommands.RenderHandler, subcommands.Params{
			ConfigPath: configPath,
			RenderFlag: renderFlag,
		}
	}
	if reloadFlag {
		return subcommands.ReloadHandler, subcommands.Params{
			ConfigPath: configPath,
		}
	}
	if maintFlag != "" {
		return subcommands.MaintenanceHandler, subcommands.Params{
			ConfigPath:      configPath,
			MaintenanceFlag: maintFlag,
		}
	}
	if putEnvFlags.Len() != 0 {
		return subcommands.PutEnvHandler, subcommands.Params{
			ConfigPath: configPath,
			Env:        putEnvFlags.Values,
		}
	}
	if putMetricFlags.Len() != 0 {
		return subcommands.PutMetricsHandler, subcommands.Params{
			ConfigPath: configPath,
			Metrics:    putMetricFlags.Values,
		}
	}
	if pingFlag {
		return subcommands.GetPingHandler, subcommands.Params{
			ConfigPath: configPath,
		}
	}

	return nil, subcommands.Params{ConfigPath: configPath}
}
