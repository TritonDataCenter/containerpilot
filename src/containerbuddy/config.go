package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
)

var (
	Version string // version for this build, set at build time via LDFLAGS
	GitHash string // short-form hash of the commit of this build, set at build time
)

type Config struct {
	Consul       string `json:"consul,omitempty"`
	OnStart      string `json:"onStart"`
	PreStop      string `json:"preStop"`
	PostStop     string `json:"postStop"`
	StopTimeout  int    `json:"stopTimeout"`
	Command      *exec.Cmd
	QuitChannels []chan bool
	Services     []*ServiceConfig `json:"services"`
	Backends     []*BackendConfig `json:"backends"`
}

type ServiceConfig struct {
	Id               string
	Name             string   `json:"name"`
	Poll             int      `json:"poll"` // time in seconds
	HealthCheckExec  string   `json:"health"`
	Port             int      `json:"port"`
	TTL              int      `json:"ttl"`
	Interfaces       []string `json:"interfaces"`
	discoveryService DiscoveryService
	ipAddress        string
}

type BackendConfig struct {
	Name             string `json:"name"`
	Poll             int    `json:"poll"` // time in seconds
	OnChangeExec     string `json:"onChange"`
	discoveryService DiscoveryService
	lastState        interface{}
}

type Pollable interface {
	PollTime() int
}

func (b BackendConfig) PollTime() int {
	return b.Poll
}
func (b *BackendConfig) CheckForUpstreamChanges() bool {
	return b.discoveryService.CheckForUpstreamChanges(b)
}

func (s ServiceConfig) PollTime() int {
	return s.Poll
}
func (s *ServiceConfig) SendHeartbeat() {
	s.discoveryService.SendHeartbeat(s)
}

func (s *ServiceConfig) MarkForMaintenance() {
	s.discoveryService.MarkForMaintenance(s)
}

func (s *ServiceConfig) Deregister() {
	s.discoveryService.Deregister(s)
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

func loadConfig() (*Config, error) {

	var configFlag string
	var versionFlag bool
	var discovery DiscoveryService
	discoveryCount := 0
	flag.StringVar(&configFlag, "config", "", "JSON config or file:// path to JSON config file.")
	flag.BoolVar(&versionFlag, "version", false, "Show version identifier and quit.")
	flag.Parse()
	if versionFlag {
		fmt.Printf("Version: %s\nGitHash: %s\n", Version, GitHash)
		os.Exit(0)
	}
	if configFlag == "" {
		configFlag = os.Getenv("CONTAINERBUDDY")
	}

	config, err := parseConfig(configFlag)
	if err != nil {
		return nil, err
	}

	for _, discoveryBackend := range []string{"Consul"} {
		switch discoveryBackend {
		case "Consul":
			if config.Consul != "" {
				discovery = NewConsulConfig(config.Consul)
				discoveryCount += 1
			}
		}
	}

	if discoveryCount == 0 {
		return nil, errors.New("No discovery backend defined")
	} else if discoveryCount > 1 {
		return nil, errors.New("More than one discovery backend defined")
	}

	if config.StopTimeout == 0 {
		config.StopTimeout = defaultStopTimeout
	}

	for _, backend := range config.Backends {
		backend.discoveryService = discovery
	}

	hostname, _ := os.Hostname()
	for _, service := range config.Services {
		service.Id = fmt.Sprintf("%s-%s", service.Name, hostname)
		service.discoveryService = discovery
		if service.ipAddress, err = getIp(service.Interfaces); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func parseConfig(configFlag string) (*Config, error) {
	if configFlag == "" {
		return nil, errors.New("-config flag is required.")
	}

	var data []byte
	if strings.HasPrefix(configFlag, "file://") {
		var err error
		fName := strings.SplitAfter(configFlag, "file://")[1]
		if data, err = ioutil.ReadFile(fName); err != nil {
			return nil, errors.New(
				fmt.Sprintf("Could not read config file: %s", err))
		}
	} else {
		data = []byte(configFlag)
	}

	config := &Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.New(fmt.Sprintf(
			"Could not parse configuration: %s",
			err))
	}

	return config, nil
}

// determine the IP address of the container
func getIp(interfaceNames []string) (string, error) {

	if interfaceNames == nil || len(interfaceNames) == 0 {
		// Use a sane default
		interfaceNames = []string{"eth0"}
	}
	interfaces := getInterfaceIps()

	// Find the interface matching the name given
	for _, interfaceName := range interfaceNames {
		for _, intf := range interfaces {
			if interfaceName == intf.Name {
				return intf.IP, nil
			}
		}
	}

	// Interface not found, return error
	return "", errors.New(fmt.Sprintf("Unable to find interfaces %s in %#v",
		interfaceNames, interfaces))
}

type InterfaceIp struct {
	Name string
	IP   string
}

func getInterfaceIps() []InterfaceIp {
	var ifaceIps []InterfaceIp
	interfaces, _ := net.Interfaces()
	for _, intf := range interfaces {
		ipAddrs, _ := intf.Addrs()
		// We're assuming each interface has one IP here because neither Docker
		// nor Triton sets up IP aliasing.
		ipAddr, _, _ := net.ParseCIDR(ipAddrs[0].String())
		ifaceIp := InterfaceIp{Name: intf.Name, IP: ipAddr.String()}
		ifaceIps = append(ifaceIps, ifaceIp)
	}
	return ifaceIps
}
