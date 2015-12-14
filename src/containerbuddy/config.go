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
	Consul       string           `json:"consul,omitempty"`
	OnStart      json.RawMessage  `json:"onStart,omitempty"`
	PreStop      json.RawMessage  `json:"preStop,omitempty"`
	PostStop     json.RawMessage  `json:"postStop,omitempty"`
	StopTimeout  int              `json:"stopTimeout"`
	Services     []*ServiceConfig `json:"services"`
	Backends     []*BackendConfig `json:"backends"`
	onStartCmd   *exec.Cmd
	preStopCmd   *exec.Cmd
	postStopCmd  *exec.Cmd
	Command      *exec.Cmd
	QuitChannels []chan bool
}

type ServiceConfig struct {
	Id               string
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	HealthCheckExec  json.RawMessage `json:"health"`
	Port             int             `json:"port"`
	TTL              int             `json:"ttl"`
	Interfaces       json.RawMessage `json:"interfaces"`
	discoveryService DiscoveryService
	ipAddress        string
	healthCheckCmd   *exec.Cmd
}

type BackendConfig struct {
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	OnChangeExec     json.RawMessage `json:"onChange"`
	discoveryService DiscoveryService
	lastState        interface{}
	onChangeCmd      *exec.Cmd
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

func (b *BackendConfig) OnChange() (int, error) {
	exitCode, err := run(b.onChangeCmd)
	// Reset command object - since it can't be reused
	b.onChangeCmd = argsToCmd(b.onChangeCmd.Args)
	return exitCode, err
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

func (s *ServiceConfig) CheckHealth() (int, error) {
	exitCode, err := run(s.healthCheckCmd)
	// Reset command object - since it can't be reused
	s.healthCheckCmd = argsToCmd(s.healthCheckCmd.Args)
	return exitCode, err
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

func parseInterfaces(raw json.RawMessage) ([]string, error) {
	if raw == nil {
		return []string{}, nil
	}
	// Parse as a string
	var jsonString string
	if err := json.Unmarshal(raw, &jsonString); err == nil {
		return []string{jsonString}, nil
	}

	var jsonArray []string
	if err := json.Unmarshal(raw, &jsonArray); err == nil {
		return jsonArray, nil
	}

	return []string{}, errors.New("interfaces must be a string or an array")
}

func parseCommandArgs(raw json.RawMessage) (*exec.Cmd, error) {
	if raw == nil {
		return nil, nil
	}
	// Parse as a string
	var stringCmd string
	if err := json.Unmarshal(raw, &stringCmd); err == nil {
		return strToCmd(stringCmd), nil
	}

	var arrayCmd []string
	if err := json.Unmarshal(raw, &arrayCmd); err == nil {
		return argsToCmd(arrayCmd), nil
	}
	return nil, errors.New("Command argument must be a string or an array")
}

func loadConfig() (*Config, error) {

	var configFlag string
	var versionFlag bool

	if !flag.Parsed() {
		flag.StringVar(&configFlag, "config", "",
			"JSON config or file:// path to JSON config file.")
		flag.BoolVar(&versionFlag, "version", false, "Show version identifier and quit.")
		flag.Parse()
	} else {
		// allows for safe configuration reload
		configFlag = flag.Lookup("config").Value.String()
	}
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
	return initializeConfig(config)
}

func initializeConfig(config *Config) (*Config, error) {
	var discovery DiscoveryService
	discoveryCount := 0
	onStartCmd, err := parseCommandArgs(config.OnStart)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `onStart`: %s", err)
	}
	config.onStartCmd = onStartCmd

	preStopCmd, err := parseCommandArgs(config.PreStop)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `preStop`: %s", err)
	}
	config.preStopCmd = preStopCmd

	postStopCmd, err := parseCommandArgs(config.PostStop)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `postStop`: %s", err)
	}
	config.postStopCmd = postStopCmd

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
		if backend.Name == "" {
			return nil, fmt.Errorf("backend must have a `name`")
		}
		cmd, err := parseCommandArgs(backend.OnChangeExec)
		if err != nil {
			return nil, fmt.Errorf("Could not parse `onChange` in backend %s: %s",
				backend.Name, err)
		}
		if cmd == nil {
			return nil, fmt.Errorf("`onChange` is required in backend %s",
				backend.Name)
		}
		if backend.Poll < 1 {
			return nil, fmt.Errorf("`poll` must be > 0 in backend %s",
				backend.Name)
		}
		backend.onChangeCmd = cmd
		backend.discoveryService = discovery
	}

	hostname, _ := os.Hostname()
	for _, service := range config.Services {
		if service.Name == "" {
			return nil, fmt.Errorf("service must have a `name`")
		}
		service.Id = fmt.Sprintf("%s-%s", service.Name, hostname)
		service.discoveryService = discovery
		if service.Poll < 1 {
			return nil, fmt.Errorf("`poll` must be > 0 in service %s",
				service.Name)
		}
		if service.TTL < 1 {
			return nil, fmt.Errorf("`ttl` must be > 0 in service %s",
				service.Name)
		}
		if service.Port < 1 {
			return nil, fmt.Errorf("`port` must be > 0 in service %s",
				service.Name)
		}

		if cmd, err := parseCommandArgs(service.HealthCheckExec); err != nil {
			return nil, fmt.Errorf("Could not parse `health` in service %s: %s",
				service.Name, err)
		} else if cmd == nil {
			return nil, fmt.Errorf("`health` is required in service %s",
				service.Name)
		} else {
			service.healthCheckCmd = cmd
		}

		interfaces, ifaceErr := parseInterfaces(service.Interfaces)
		if ifaceErr != nil {
			return nil, ifaceErr
		}

		if service.ipAddress, err = getIp(interfaces); err != nil {
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
	return unmarshalConfig(data)
}

func unmarshalConfig(data []byte) (*Config, error) {
	config := &Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(
			"Could not parse configuration: %s",
			err)
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

func argsToCmd(args []string) *exec.Cmd {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return exec.Command(args[0], args[1:]...)
	} else {
		return exec.Command(args[0])
	}
}

func strToCmd(command string) *exec.Cmd {
	if command != "" {
		return argsToCmd(strings.Split(command, " "))
	}
	return nil
}
