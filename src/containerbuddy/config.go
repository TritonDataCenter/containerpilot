package main

import (
	"flag"
	"strings"
)

type Config struct {
	DiscoveryService DiscoveryService
	pollTime         int
	healthCheckExec  string
	serviceName      string
	onChangeExec     string
	toCheck          arrayFlags
}

// type alias to deal with parsing multiple -check params
type arrayFlags []string

func (i *arrayFlags) String() string { return strings.Join([]string(*i), ",") }
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func parseArgs() *Config {
	var (
		discoveryUri    = flag.String("consul", "consul:8500", "Hostname and port for consul.")
		pollTime        = flag.Int("poll", 10, "Number of seconds to wait between polling health check")
		healthCheckExec = flag.String("health", "", "Executable to run to check the health of the application.")
		serviceName     = flag.String("name", "", "Name of service to register.")
		onChangeExec    = flag.String("onChange", "", "Executable to run when the discovery service has changes.")
		toCheck         arrayFlags
	)
	flag.Var(&toCheck, "check", "What services to check for changes (accepts multiple).")
	flag.Parse()

	config := &Config{
		DiscoveryService: NewConsulConfig(*discoveryUri, *serviceName, *pollTime, toCheck),
		pollTime:         *pollTime,
		healthCheckExec:  *healthCheckExec,
		onChangeExec:     *onChangeExec,
	}
	return config
}
