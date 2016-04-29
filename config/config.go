package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/joyent/containerpilot/backends"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/tasks"
	"github.com/joyent/containerpilot/telemetry"
)

// Config is the top-level ContainerPilot Configuration
type Config struct {
	Consul          interface{}   `json:"consul"`
	Etcd            interface{}   `json:"etcd"`
	LogConfig       *LogConfig    `json:"logging"`
	OnStart         interface{}   `json:"onStart"`
	PreStart        interface{}   `json:"preStart"`
	PreStop         interface{}   `json:"preStop"`
	PostStop        interface{}   `json:"postStop"`
	StopTimeout     int           `json:"stopTimeout"`
	ServicesConfig  []interface{} `json:"services"`
	BackendsConfig  []interface{} `json:"backends"`
	TasksConfig     []interface{} `json:"tasks"`
	TelemetryConfig interface{}   `json:"telemetry"`

	ConfigFlag string
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

// ParseDiscoveryService ...
func (cfg *Config) ParseDiscoveryService() (discovery.DiscoveryService, error) {
	var discoveryService discovery.DiscoveryService
	var err error
	discoveryCount := 0
	for _, discoveryBackend := range []string{"Consul", "Etcd"} {
		switch discoveryBackend {
		case "Consul":
			if cfg.Consul != nil {
				discoveryService, err = discovery.NewConsulConfig(cfg.Consul)
				if err != nil {
					return nil, err
				}
				discoveryCount++
			}
		case "Etcd":
			if cfg.Etcd != nil {
				discoveryService, err = discovery.NewEtcdConfig(cfg.Etcd)
				if err != nil {
					return nil, err
				}
				discoveryCount++
			}
		}
	}
	if discoveryCount == 0 {
		return nil, errors.New("No discovery backend defined")
	} else if discoveryCount > 1 {
		return nil, errors.New("More than one discovery backend defined")
	}
	return discoveryService, nil
}

// InitLogging ...
func (cfg *Config) InitLogging() error {
	if cfg.LogConfig != nil {
		return cfg.LogConfig.init()
	}
	return nil
}

// ParseBackends ...
func (cfg *Config) ParseBackends(discoveryService discovery.DiscoveryService) ([]*backends.Backend, error) {
	backends, err := backends.NewBackends(cfg.BackendsConfig, discoveryService)
	if err != nil {
		return nil, err
	}
	return backends, nil
}

// ParseServices ...
func (cfg *Config) ParseServices(discoveryService discovery.DiscoveryService) ([]*services.Service, error) {
	services, err := services.NewServices(cfg.ServicesConfig, discoveryService)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// ParseStopTimeout ...
func (cfg *Config) ParseStopTimeout() (int, error) {
	if cfg.StopTimeout == 0 {
		return defaultStopTimeout, nil
	}
	return cfg.StopTimeout, nil
}

// ParseTelemetry ...
func (cfg *Config) ParseTelemetry() (*telemetry.Telemetry, error) {
	if cfg.TelemetryConfig == nil {
		return nil, nil
	}
	t, err := telemetry.NewTelemetry(cfg.TelemetryConfig)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// CreateTelemetryService ...
func CreateTelemetryService(t *telemetry.Telemetry, discoveryService discovery.DiscoveryService) (*services.Service, error) {
	// create a new service for Telemetry
	svc, err := services.NewService(
		t.ServiceName,
		t.Poll,
		t.Port,
		t.TTL,
		t.Interfaces,
		t.Tags,
		discoveryService)
	if err != nil {
		return nil, err
	}
	return svc, nil
}

// ParseTasks ...
func (cfg *Config) ParseTasks() ([]*tasks.Task, error) {
	if cfg.TasksConfig == nil {
		return nil, nil
	}
	tasks, err := tasks.NewTasks(cfg.TasksConfig)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// ParseConfig ...
func ParseConfig(configFlag string) (*Config, error) {
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
	cfg, err := UnmarshalConfig(template)
	if cfg != nil {
		// store so we can reload
		cfg.ConfigFlag = configFlag
	}
	return cfg, err
}

// UnmarshalConfig unmarshalls the raw config bytes into a Config struct
func UnmarshalConfig(data []byte) (*Config, error) {
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
