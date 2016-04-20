package config

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

	"github.com/joyent/containerpilot/backends"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/utils"
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
// accessing via `GetConfig` at the "top" of a goroutine and then use the config
// as context for a function after that.
var (
	globalConfig *Config
	configLock   = new(sync.RWMutex)
)

func GetConfig() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return globalConfig
}

// Config is the top-level ContainerPilot Configuration
type Config struct {
	Consul          string          `json:"consul,omitempty"`
	Etcd            json.RawMessage `json:"etcd,omitempty"`
	LogConfig       *LogConfig      `json:"logging,omitempty"`
	OnStart         json.RawMessage `json:"onStart,omitempty"`
	PreStart        json.RawMessage `json:"preStart,omitempty"`
	PreStop         json.RawMessage `json:"preStop,omitempty"`
	PostStop        json.RawMessage `json:"postStop,omitempty"`
	StopTimeout     int             `json:"stopTimeout"`
	ServicesConfig  json.RawMessage `json:"services,omitempty"`
	BackendsConfig  json.RawMessage `json:"backends,omitempty"`
	TelemetryConfig json.RawMessage `json:"telemetry,omitempty"`
	Services        []*services.Service
	Backends        []*backends.Backend
	Telemetry       *telemetry.Telemetry
	PreStartCmd     *exec.Cmd
	PreStopCmd      *exec.Cmd
	PostStopCmd     *exec.Cmd
	Command         *exec.Cmd
	QuitChannels    []chan bool
	ConfigFlag      string
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

func LoadConfig() (*Config, error) {

	var configFlag string
	var versionFlag bool

	if !flag.Parsed() {
		flag.StringVar(&configFlag, "config", "",
			"JSON config or file:// path to JSON config file.")
		flag.BoolVar(&versionFlag, "version", false, "Show version identifier and quit.")
		flag.Parse()
	}
	if versionFlag {
		fmt.Printf("Version: %s\nGitHash: %s\n", Version, GitHash)
		os.Exit(0)
	}
	if configFlag == "" {
		configFlag = os.Getenv("CONTAINERPILOT")
	}

	if cfg, err := parseConfig(configFlag); err != nil {
		return nil, err
	} else {
		return initializeConfig(cfg)
	}
}

func ReloadConfig(configFlag string) (*Config, error) {
	if cfg, err := parseConfig(configFlag); err != nil {
		return nil, err
	} else {
		return initializeConfig(cfg)
	}
}

func initializeConfig(cfg *Config) (*Config, error) {
	var discoveryService discovery.DiscoveryService
	discoveryCount := 0

	// onStart has been deprecated for preStart. Remove in 2.0
	if cfg.PreStart != nil && cfg.OnStart != nil {
		fmt.Println("The onStart option has been deprecated in favor of preStart. ContainerPilot will use only the preStart option provided")
	}

	// alias the onStart behavior to preStart
	if cfg.PreStart == nil && cfg.OnStart != nil {
		fmt.Println("The onStart option has been deprecated in favor of preStart. ContainerPilot will use the onStart option as a preStart")
		cfg.PreStart = cfg.OnStart
	}

	preStartCmd, err := utils.ParseCommandArgs(cfg.PreStart)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `preStart`: %s", err)
	}
	cfg.PreStartCmd = preStartCmd

	preStopCmd, err := utils.ParseCommandArgs(cfg.PreStop)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `preStop`: %s", err)
	}
	cfg.PreStopCmd = preStopCmd

	postStopCmd, err := utils.ParseCommandArgs(cfg.PostStop)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `postStop`: %s", err)
	}
	cfg.PostStopCmd = postStopCmd

	for _, discoveryBackend := range []string{"Consul", "Etcd"} {
		switch discoveryBackend {
		case "Consul":
			if cfg.Consul != "" {
				discoveryService = discovery.NewConsulConfig(cfg.Consul)
				discoveryCount++
			}
		case "Etcd":
			if cfg.Etcd != nil {
				discoveryService = discovery.NewEtcdConfig(cfg.Etcd)
				discoveryCount++
			}
		}
	}

	if discoveryCount == 0 {
		return nil, errors.New("No discovery backend defined")
	} else if discoveryCount > 1 {
		return nil, errors.New("More than one discovery backend defined")
	}

	if cfg.LogConfig != nil {
		err := cfg.LogConfig.init()
		if err != nil {
			return nil, err
		}
	}

	if cfg.StopTimeout == 0 {
		cfg.StopTimeout = defaultStopTimeout
	}

	if backends, err := backends.NewBackends(cfg.BackendsConfig,
		discoveryService); err != nil {
		return nil, err
	} else {
		cfg.Backends = backends
	}

	if services, err := services.NewServices(cfg.ServicesConfig,
		discoveryService); err != nil {
		return nil, err
	} else {
		cfg.Services = services
	}

	if cfg.TelemetryConfig != nil {
		if t, err := telemetry.NewTelemetry(cfg.TelemetryConfig); err != nil {
			return nil, err
		} else {
			cfg.Telemetry = t
			// create a new service for Telemetry
			if telemetryService, err := services.NewService(
				t.ServiceName,
				t.Poll,
				t.Port,
				t.TTL,
				t.Interfaces,
				t.Tags,
				discoveryService); err != nil {
				return nil, err
			} else {
				cfg.Services = append(cfg.Services, telemetryService)
			}
		}
	}

	configLock.Lock()
	globalConfig = cfg
	configLock.Unlock()

	return cfg, nil
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
	cfg, err := unmarshalConfig(template)
	if cfg != nil {
		// store so we can reload
		cfg.ConfigFlag = configFlag
	}
	return cfg, err
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
			count := int(7 + col - 1)
			if count > 0 {
				highlight = fmt.Sprintf("%s^", strings.Repeat("-", count))
			}
			break
		}
		col -= readBytes + 1
		line++
	}
	if col < 0 {
		highlight = "Do you have an extra comma somewhere?"
	}
	return line, int(col), fmt.Sprintf("%s%s%s", prevLine, thisLine, highlight)
}
