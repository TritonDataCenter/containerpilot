package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

type Config struct {
	Consul      string `json:"consul,omitempty"`
	OnStart     string `json:"onStart"`
	onStartArgs []string
	Services    []*ServiceConfig `json:"services"`
	Backends    []*BackendConfig `json:"backends"`
}

type ServiceConfig struct {
	Id               string
	Name             string `json:"name"`
	Poll             int    `json:"poll"` // time in seconds
	HealthCheckExec  string `json:"health"`
	Port             int    `json:"port"`
	TTL              int    `json:"ttl"`
	IsPublic         bool   `json:"publicIp"` // will default to false
	Interface        string `json:"interface"`
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
		service.ipAddress = getIp(service.IsPublic,service.Interface)
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

// determine the IP address of the container
func getIp(usePublic bool, interfaceName string) string {
	interfaces, _ := net.Interfaces()
	var ips []net.IP
	_, loopback, _ := net.ParseCIDR("127.0.0.0/8")
	for _, intf := range interfaces {
		ipAddrs, _ := intf.Addrs()
		// We're assuming each interface has one IP here because neither Docker
		// nor Triton sets up IP aliasing.
		ipAddr, _, _ := net.ParseCIDR(ipAddrs[0].String())
		// Use specific interface if given
		if interfaceName == intf.Name {
			return ipAddr.String()
		}
		if !loopback.Contains(ipAddr) {
			ips = append(ips, ipAddr)
		}
	}
	var ip string
	for _, ipAddr := range ips {
		isPublic := isPublicIp(ipAddr)
		if isPublic && usePublic {
			ip = ipAddr.String()
			break
		} else if !isPublic && !usePublic {
			ip = ipAddr.String()
			break
		}
	}
	return ip
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
