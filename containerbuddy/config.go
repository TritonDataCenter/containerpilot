package containerbuddy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	// Version is the version for this build, set at build time via LDFLAGS
	Version string
	// GitHash is the short-form commit hash of this build, set at build time
	GitHash string
)

// Passing around config as a context to functions would be the ideomatic way.
// But we need to support configuration reload from signals and have that reload
// effect function calls in the main goroutine. Wherever possible we should be
// accessing via `getConfig` at the "top" of a goroutine and then use the config
// as context for a function after that.
var (
	globalConfig *Config
	configLock   = new(sync.RWMutex)
)

func getConfig() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return globalConfig
}

// Config is the top-level Containerbuddy Configuration
type Config struct {
	Consul       string           `json:"consul,omitempty"`
	Etcd         json.RawMessage  `json:"etcd,omitempty"`
	LogConfig    *LogConfig       `json:"logging,omitempty"`
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

// ServiceConfig configures the service, discovery data, and health checks
type ServiceConfig struct {
	ID               string
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	HealthCheckExec  json.RawMessage `json:"health"`
	Port             int             `json:"port"`
	TTL              int             `json:"ttl"`
	Interfaces       json.RawMessage `json:"interfaces"`
	Tags             []string        `json:"tags,omitempty"`
	Address          string          `json:"address,omitempty"`
	discoveryService DiscoveryService
	ipAddress        string
	healthCheckCmd   *exec.Cmd
}

// BackendConfig represents a command to execute when another application changes
type BackendConfig struct {
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	OnChangeExec     json.RawMessage `json:"onChange"`
	Tag              string          `json:"tag,omitempty"`
	discoveryService DiscoveryService
	lastState        interface{}
	onChangeCmd      *exec.Cmd
}

// Pollable is base abstraction for backends and services that support polling
type Pollable interface {
	PollTime() int
}

// PollTime returns the backend's  poll time
func (b BackendConfig) PollTime() int {
	return b.Poll
}

// CheckForUpstreamChanges checks the service discovery endpoint for any changes
// in a dependent backend. Returns true when there has been a change.
func (b *BackendConfig) CheckForUpstreamChanges() bool {
	return b.discoveryService.CheckForUpstreamChanges(b)
}

// OnChange runs the backend's onChange command, returning the results
func (b *BackendConfig) OnChange() (int, error) {
	exitCode, err := run(b.onChangeCmd)
	// Reset command object - since it can't be reused
	b.onChangeCmd = argsToCmd(b.onChangeCmd.Args)
	return exitCode, err
}

// PollTime returns the service's poll time
func (s ServiceConfig) PollTime() int {
	return s.Poll
}

// SendHeartbeat sends a heartbeat for this service
func (s *ServiceConfig) SendHeartbeat() {
	s.discoveryService.SendHeartbeat(s)
}

// MarkForMaintenance marks this service for maintenance
func (s *ServiceConfig) MarkForMaintenance() {
	s.discoveryService.MarkForMaintenance(s)
}

// Deregister will deregister this instance of the service
func (s *ServiceConfig) Deregister() {
	s.discoveryService.Deregister(s)
}

// CheckHealth runs the service's health command, returning the results
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

	for _, discoveryBackend := range []string{"Consul", "Etcd"} {
		switch discoveryBackend {
		case "Consul":
			if config.Consul != "" {
				discovery = NewConsulConfig(config.Consul)
				discoveryCount++
			}
		case "Etcd":
			if config.Etcd != nil {
				discovery = NewEtcdConfig(config.Etcd)
				discoveryCount++
			}
		}
	}

	if discoveryCount == 0 {
		return nil, errors.New("No discovery backend defined")
	} else if discoveryCount > 1 {
		return nil, errors.New("More than one discovery backend defined")
	}

	if config.LogConfig != nil {
		err := config.LogConfig.init()
		if err != nil {
			return nil, err
		}
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
		service.ID = fmt.Sprintf("%s-%s", service.Name, hostname)
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

		if service.Address != "" {
			service.ipAddress = service.Address
		} else {
			if service.ipAddress, err = GetIP(interfaces); err != nil {
				return nil, err
			}
		}
	}

	configLock.Lock()
	globalConfig = config
	configLock.Unlock()

	return config, nil
}

func parseConfig(configFlag string) (*Config, error) {
	if configFlag == "" {
		return nil, errors.New("-config flag is required")
	}

	var data []byte
	if strings.HasPrefix(configFlag, "file://") {
		var err error
		fName := strings.SplitAfter(configFlag, "file://")[1]
		if data, err = ioutil.ReadFile(fName); err != nil {
			return nil, fmt.Errorf("Could not read config file: %s", err)
		}
	} else {
		data = []byte(configFlag)
	}

	template, err := ApplyTemplate(data)
	if err != nil {
		return nil, fmt.Errorf(
			"Could not apply template to config: %s", err)
	}
	return unmarshalConfig(template)
}

func unmarshalConfig(data []byte) (*Config, error) {
	config := &Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		syntax, ok := err.(*json.SyntaxError)
		if !ok {
			return nil, fmt.Errorf(
				"Could not parse configuration: %s",
				err)
		}
		return nil, newJSONParseError(data, syntax)
	}
	return config, nil
}

func newJSONParseError(js []byte, syntax *json.SyntaxError) error {
	line, col, err := highlightError(js, syntax.Offset)
	return fmt.Errorf("Parse error at line:col [%d:%d]: %s\n%s", line, col, syntax, err)
}

func highlightError(data []byte, pos int64) (int, int, string) {
	prevLine := ""
	thisLine := ""
	highlight := ""
	line := 1
	col := pos
	offset := int64(0)
	r := bytes.NewReader(data)
	scanner := bufio.NewScanner(r)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		prevLine = thisLine
		thisLine = fmt.Sprintf("%5d: %s\n", line, scanner.Text())
		readBytes := int64(len(scanner.Bytes()))
		offset += readBytes
		if offset >= pos-1 {
			highlight = fmt.Sprintf("%s^", strings.Repeat("-", int(7+col-1)))
			break
		}
		col -= readBytes + 1
		line++
	}
	return line, int(col), fmt.Sprintf("%s%s%s", prevLine, thisLine, highlight)
}

func argsToCmd(args []string) *exec.Cmd {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return exec.Command(args[0], args[1:]...)
	}
	return exec.Command(args[0])
}

func strToCmd(command string) *exec.Cmd {
	if command != "" {
		return argsToCmd(strings.Split(strings.TrimSpace(command), " "))
	}
	return nil
}
