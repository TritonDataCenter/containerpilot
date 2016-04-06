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
	"telemetry"
	"utils"
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
	Consul       string               `json:"consul,omitempty"`
	Etcd         json.RawMessage      `json:"etcd,omitempty"`
	LogConfig    *LogConfig           `json:"logging,omitempty"`
	OnStart      json.RawMessage      `json:"onStart,omitempty"`
	PreStart     json.RawMessage      `json:"preStart,omitempty"`
	PreStop      json.RawMessage      `json:"preStop,omitempty"`
	PostStop     json.RawMessage      `json:"postStop,omitempty"`
	StopTimeout  int                  `json:"stopTimeout"`
	Services     []*ServiceConfig     `json:"services"`
	Backends     []*BackendConfig     `json:"backends"`
	Telemetry    *telemetry.Telemetry `json:"telemetry,omitempty"`
	preStartCmd  *exec.Cmd
	preStopCmd   *exec.Cmd
	postStopCmd  *exec.Cmd
	Command      *exec.Cmd
	QuitChannels []chan bool
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

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

	// onStart has been deprecated for preStart. Remove in 2.0
	if config.PreStart != nil && config.OnStart != nil {
		fmt.Println("The onStart option has been deprecated in favor of preStart. Containerbuddy will use only the preStart option provided")
	}

	// alias the onStart behavior to preStart
	if config.PreStart == nil && config.OnStart != nil {
		fmt.Println("The onStart option has been deprecated in favor of preStart. Containerbuddy will use the onStart option as a preStart")

		//		"preStart is the executable (and its arguments) that will be called immediately prior to starting the shimmed application. If the preStart handler returns a non-zero exit code, Containerbuddy will exit."
		config.PreStart = config.OnStart
	}

	preStartCmd, err := utils.ParseCommandArgs(config.PreStart)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `preStart`: %s", err)
	}
	config.preStartCmd = preStartCmd

	preStopCmd, err := utils.ParseCommandArgs(config.PreStop)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `preStop`: %s", err)
	}
	config.preStopCmd = preStopCmd

	postStopCmd, err := utils.ParseCommandArgs(config.PostStop)
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
		cmd, err := utils.ParseCommandArgs(backend.OnChangeExec)
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

		if cmd, err := utils.ParseCommandArgs(service.HealthCheckExec); err != nil {
			return nil, fmt.Errorf("Could not parse `health` in service %s: %s",
				service.Name, err)
		} else if cmd == nil {
			return nil, fmt.Errorf("`health` is required in service %s",
				service.Name)
		} else {
			service.healthCheckCmd = cmd
		}

		if ipAddress, err := utils.IpFromInterfaces(service.Interfaces); err != nil {
			return nil, err
		} else {
			service.ipAddress = ipAddress
		}
	}

	if config.Telemetry != nil {
		m := config.Telemetry
		if err := m.Parse(); err != nil {
			return nil, err
		} else {
			// create a new service for Telemetry
			telemetryService := &ServiceConfig{
				ID:               fmt.Sprintf("%s-%s", m.ServiceName, hostname),
				Name:             m.ServiceName,
				Poll:             m.Poll,
				Port:             m.Port,
				TTL:              m.TTL,
				Interfaces:       m.Interfaces,
				Tags:             m.Tags,
				discoveryService: discovery,
				ipAddress:        m.IpAddress,
				healthCheckCmd:   nil, // no health check code
			}
			config.Services = append(config.Services, telemetryService)
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
