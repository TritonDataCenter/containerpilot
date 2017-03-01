package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/backends"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/coprocesses"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/tasks"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/socket"
	"github.com/joyent/containerpilot/utils"
)

type rawConfig struct {
	logConfig         *LogConfig
	preStart          interface{}
	preStop           interface{}
	postStop          interface{}
	stopTimeout       int
	coprocessesConfig []interface{}
	servicesConfig    []interface{}
	backendsConfig    []interface{}
	tasksConfig       []interface{}
	telemetryConfig   interface{}
	socketConfig      interface{}
}

// Config contains the parsed config elements
type Config struct {
	ServiceBackend discovery.ServiceBackend
	LogConfig      *LogConfig
	PreStart       *commands.Command
	PreStop        *commands.Command
	PostStop       *commands.Command
	StopTimeout    int
	Coprocesses    []*coprocesses.Coprocess
	Services       []*services.Service
	Backends       []*backends.Backend
	Tasks          []*tasks.Task
	Telemetry      *telemetry.Telemetry
	ControlSocket  *socket.ControlSocket
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

func parseServiceBackend(rawCfg map[string]interface{}) (discovery.ServiceBackend, error) {
	var discoveryService discovery.ServiceBackend
	var err error
	discoveryCount := 0
	for _, key := range discovery.GetBackends() {
		handler := discovery.GetConfigHook(key)
		if handler != nil {
			if rawCfg, ok := rawCfg[key]; ok {
				discoveryService, err = handler(rawCfg)
				if err != nil {
					return nil, err
				}
				log.Debugf("parsed service discovery backend: %s", key)
				discoveryCount++
			}
			delete(rawCfg, key)
		}
	}
	if discoveryCount == 0 {
		return nil, errors.New("No discovery backend defined")
	} else if discoveryCount > 1 {
		return nil, errors.New("More than one discovery backend defined")
	}
	return discoveryService, nil
}

// InitLogging configure logrus with the new log config if available
func (cfg *Config) InitLogging() error {
	if cfg.LogConfig != nil {
		return cfg.LogConfig.init()
	}
	return nil
}

func (cfg *rawConfig) parseBackends(discoveryService discovery.ServiceBackend) ([]*backends.Backend, error) {
	backends, err := backends.NewBackends(cfg.backendsConfig, discoveryService)
	if err != nil {
		return nil, err
	}
	return backends, nil
}

func (cfg *rawConfig) parseServices(discoveryService discovery.ServiceBackend) ([]*services.Service, error) {
	services, err := services.NewServices(cfg.servicesConfig, discoveryService)
	if err != nil {
		return nil, err
	}
	return services, nil
}

func (cfg *rawConfig) parseCoprocesses() ([]*coprocesses.Coprocess, error) {
	coprocesses, err := coprocesses.NewCoprocesses(cfg.coprocessesConfig)
	if err != nil {
		return nil, err
	}
	return coprocesses, nil
}

// parseStopTimeout ...
func (cfg *rawConfig) parseStopTimeout() (int, error) {
	if cfg.stopTimeout == 0 {
		return defaultStopTimeout, nil
	}
	return cfg.stopTimeout, nil
}

// parseTelemetry ...
func (cfg *rawConfig) parseTelemetry() (*telemetry.Telemetry, error) {

	if cfg.telemetryConfig == nil {
		return nil, nil
	}
	t, err := telemetry.NewTelemetry(cfg.telemetryConfig)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// createTelemetryService ...
func createTelemetryService(t *telemetry.Telemetry, discoveryService discovery.ServiceBackend) (*services.Service, error) {
	// create a new service for Telemetry
	svc, err := services.NewService(
		t.ServiceName,
		t.Poll,
		t.Port,
		t.TTL,
		t.Interfaces,
		t.Tags,
		nil,
		discoveryService)
	if err != nil {
		return nil, err
	}
	return svc, nil
}

func (cfg *rawConfig) parseTasks() ([]*tasks.Task, error) {
	if cfg.tasksConfig == nil {
		return nil, nil
	}
	tasks, err := tasks.NewTasks(cfg.tasksConfig)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// RenderConfig renders the templated config in configFlag to renderFlag.
func RenderConfig(configFlag, renderFlag string) error {
	template, err := renderConfigTemplate(configFlag)
	if err != nil {
		return err
	}

	// Save the template, either to stdout or to file://...
	if renderFlag == "-" {
		fmt.Printf("%s", template)
	} else if strings.HasPrefix(renderFlag, "file://") {
		var err error
		fName := strings.SplitAfter(renderFlag, "file://")[1]
		if err = ioutil.WriteFile(fName, template, 0644); err != nil {
			return fmt.Errorf("Could not write config file: %s", err)
		}
	} else {
		return fmt.Errorf("-render flag is invalid: '%s'", renderFlag)
	}

	return nil
}

// ParseConfig parses a raw config flag
func ParseConfig(configFlag string) (*Config, error) {

	template, err := renderConfigTemplate(configFlag)
	if err != nil {
		return nil, err
	}
	configMap, err := unmarshalConfig(template)
	if err != nil {
		return nil, err
	}
	discoveryService, err := parseServiceBackend(configMap)
	if err != nil {
		return nil, err
	}
	// Delete discovery backend keys
	for _, backend := range discovery.GetBackends() {
		delete(configMap, backend)
	}
	raw := &rawConfig{}
	if err = decodeConfig(configMap, raw); err != nil {
		return nil, err
	}
	cfg := &Config{}
	cfg.ServiceBackend = discoveryService
	cfg.LogConfig = raw.logConfig

	preStartCmd, err := raw.parsePreStart()
	if err != nil {
		return nil, err
	}
	cfg.PreStart = preStartCmd

	preStopCmd, err := raw.parsePreStop()
	if err != nil {
		return nil, err
	}
	cfg.PreStop = preStopCmd

	postStopCmd, err := raw.parsePostStop()
	if err != nil {
		return nil, err
	}
	cfg.PostStop = postStopCmd

	stopTimeout, err := raw.parseStopTimeout()
	if err != nil {
		return nil, err
	}
	cfg.StopTimeout = stopTimeout

	services, err := raw.parseServices(discoveryService)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse services: %v", err)
	}
	cfg.Services = services

	backends, err := raw.parseBackends(discoveryService)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse backends: %v", err)
	}
	cfg.Backends = backends

	telemetry, err := raw.parseTelemetry()
	if err != nil {
		return nil, err
	}

	if telemetry != nil {
		telemetryService, err2 := createTelemetryService(telemetry, discoveryService)
		if err2 != nil {
			return nil, err2
		}
		cfg.Telemetry = telemetry
		cfg.Services = append(cfg.Services, telemetryService)
	}

	tasks, err := raw.parseTasks()
	if err != nil {
		return nil, err
	}
	cfg.Tasks = tasks

	coprocesses, err := raw.parseCoprocesses()
	if err != nil {
		return nil, err
	}
	cfg.Coprocesses = coprocesses

	return cfg, nil
}

func renderConfigTemplate(configFlag string) ([]byte, error) {
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
		err = fmt.Errorf("Could not apply template to config: %v", err)
	}
	return template, err
}

func unmarshalConfig(data []byte) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		syntax, ok := err.(*json.SyntaxError)
		if !ok {
			return nil, fmt.Errorf(
				"Could not parse configuration: %s",
				err)
		}
		return nil, newJSONparseError(data, syntax)
	}
	return config, nil
}

func newJSONparseError(js []byte, syntax *json.SyntaxError) error {
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

func decodeArray(raw interface{}) []interface{} {
	if raw == nil {
		return nil
	}
	var arr []interface{}
	switch reflect.TypeOf(raw).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(raw)
		for i := 0; i < s.Len(); i++ {
			v := s.Index(i)
			if !v.IsNil() {
				arr = append(arr, v.Interface())
			}
		}
		return arr
	}
	return nil
}

// We can't use mapstructure to decode our config map since we want the values
// to also be raw interface{} types. mapstructure can only decode
// into concrete structs and primitives
func decodeConfig(configMap map[string]interface{}, result *rawConfig) error {
	var logConfig LogConfig
	var stopTimeout int
	if err := utils.DecodeRaw(configMap["logging"], &logConfig); err != nil {
		return err
	}
	if err := utils.DecodeRaw(configMap["stopTimeout"], &stopTimeout); err != nil {
		return err
	}
	result.stopTimeout = stopTimeout
	result.logConfig = &logConfig
	result.preStart = configMap["preStart"]
	result.preStop = configMap["preStop"]
	result.postStop = configMap["postStop"]
	result.servicesConfig = decodeArray(configMap["services"])
	result.backendsConfig = decodeArray(configMap["backends"])
	result.tasksConfig = decodeArray(configMap["tasks"])
	result.coprocessesConfig = decodeArray(configMap["coprocesses"])
	result.telemetryConfig = configMap["telemetry"]

	delete(configMap, "logging")
	delete(configMap, "preStart")
	delete(configMap, "preStop")
	delete(configMap, "postStop")
	delete(configMap, "stopTimeout")
	delete(configMap, "services")
	delete(configMap, "backends")
	delete(configMap, "tasks")
	delete(configMap, "coprocesses")
	delete(configMap, "telemetry")
	var unused []string
	for key := range configMap {
		unused = append(unused, key)
	}
	if len(unused) > 0 {
		return fmt.Errorf("Unknown config keys: %v", unused)
	}
	return nil
}
