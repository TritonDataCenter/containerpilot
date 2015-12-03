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

type Config struct {
	Consul      string `json:"consul,omitempty"`
	OnStart     string `json:"onStart"`
	StopTimeout int    `json:"stopTimeout"`
	onStartArgs []string
	Command     *exec.Cmd
	Services    []*ServiceConfig `json:"services"`
	Backends    []*BackendConfig `json:"backends"`
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
	healthArgs       []string
	ipAddress        string
}

type BackendConfig struct {
	Name             string `json:"name"`
	Poll             int    `json:"poll"` // time in seconds
	OnChangeExec     string `json:"onChange"`
	discoveryService DiscoveryService
	onChangeArgs     []string
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

func loadConfig() (*Config, error) {

	var configFlag string
	var discovery DiscoveryService
	discoveryCount := 0
	flag.StringVar(&configFlag, "config", "", "JSON config or file:// path to JSON config file.")
	flag.Parse()
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
		config.StopTimeout = 10
	}

	config.onStartArgs = strings.Split(config.OnStart, " ")

	for _, backend := range config.Backends {
		backend.discoveryService = discovery
		backend.onChangeArgs = strings.Split(backend.OnChangeExec, " ")
	}

	hostname, _ := os.Hostname()
	for _, service := range config.Services {
		service.Id = fmt.Sprintf("%s-%s", service.Name, hostname)
		service.discoveryService = discovery
		service.healthArgs = strings.Split(service.HealthCheckExec, " ")
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
		if data, err = ioutil.ReadFile(strings.SplitAfter(configFlag, "file://")[1]); err != nil {
			return nil, errors.New(fmt.Sprintf("Could not read config file: %s", err))
		}
	} else {
		data = []byte(configFlag)
	}

	config := &Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.New(fmt.Sprintf("Could not parse configuration: %s", err))
	}

	return config, nil
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
	return "", errors.New(fmt.Sprintf("Unable to find interfaces %s in %#v", interfaceNames, interfaces))
}

// parse an IPv4 address and return true if it's a public IP
func isPublicIp(ip net.IP) bool {
	_, c, _ := net.ParseCIDR("192.168.0.0/16")
	_, b, _ := net.ParseCIDR("172.16.0.0/12")
	_, a, _ := net.ParseCIDR("10.0.0.0/8")

	var privateNetworks = []*net.IPNet{c, b, a}
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}
