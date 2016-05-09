package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/backends"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/tasks"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/utils"
)

type rawConfig struct {
	logConfig       *LogConfig
	onStart         interface{}
	preStart        interface{}
	preStop         interface{}
	postStop        interface{}
	stopTimeout     int
	servicesConfig  []interface{}
	backendsConfig  []interface{}
	tasksConfig     []interface{}
	telemetryConfig interface{}
}

// Config contains the parsed config elements
type Config struct {
	DiscoveryService discovery.DiscoveryService
	LogConfig        *LogConfig
	PreStart         *exec.Cmd
	PreStop          *exec.Cmd
	PostStop         *exec.Cmd
	StopTimeout      int
	Services         []*services.Service
	Backends         []*backends.Backend
	Tasks            []*tasks.Task
	Telemetry        *telemetry.Telemetry
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

func parseDiscoveryService(rawCfg map[string]interface{}) (discovery.DiscoveryService, error) {
	var discoveryService discovery.DiscoveryService
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

func (cfg *rawConfig) parseBackends(discoveryService discovery.DiscoveryService) ([]*backends.Backend, error) {
	backends, err := backends.NewBackends(cfg.backendsConfig, discoveryService)
	if err != nil {
		return nil, err
	}
	return backends, nil
}

func (cfg *rawConfig) parseServices(discoveryService discovery.DiscoveryService) ([]*services.Service, error) {
	services, err := services.NewServices(cfg.servicesConfig, discoveryService)
	if err != nil {
		return nil, err
	}
	return services, nil
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
func createTelemetryService(t *telemetry.Telemetry, discoveryService discovery.DiscoveryService) (*services.Service, error) {
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

// ParseConfig parses a raw config flag
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
			"Could not apply template to config: %v", err)
	}
	configMap, err := unmarshalConfig(template)
	if err != nil {
		return nil, err
	}
	discoveryService, err := parseDiscoveryService(configMap)
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
	cfg.DiscoveryService = discoveryService
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

	return cfg, nil
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
	result.onStart = configMap["onStart"]
	result.preStart = configMap["preStart"]
	result.preStop = configMap["preStop"]
	result.postStop = configMap["postStop"]
	result.servicesConfig = decodeArray(configMap["services"])
	result.backendsConfig = decodeArray(configMap["backends"])
	result.tasksConfig = decodeArray(configMap["tasks"])
	result.telemetryConfig = configMap["telemetry"]

	delete(configMap, "logging")
	delete(configMap, "onStart")
	delete(configMap, "preStart")
	delete(configMap, "preStop")
	delete(configMap, "postStop")
	delete(configMap, "services")
	delete(configMap, "backends")
	delete(configMap, "tasks")
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
