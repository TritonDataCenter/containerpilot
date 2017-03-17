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
	"github.com/joyent/containerpilot/checks"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/utils"
	"github.com/joyent/containerpilot/watches"
)

type rawConfig struct {
	logConfig   *LogConfig
	stopTimeout int
	services    []interface{}
	watches     []interface{}
	telemetry   interface{}
}

// Config contains the parsed config elements
type Config struct {
	Discovery   discovery.Backend
	LogConfig   *LogConfig
	StopTimeout int
	Services    []*services.Config
	Checks      []*checks.Config
	Watches     []*watches.Config
	Telemetry   *telemetry.Config
}

const (
	// Amount of time to wait before killing the application
	defaultStopTimeout int = 5
)

func parseDiscoveryBackend(rawCfg map[string]interface{}) (discovery.Backend, error) {
	var discoveryService discovery.Backend
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
		return nil, errors.New("no discovery backend defined")
	} else if discoveryCount > 1 {
		return nil, errors.New("more than one discovery backend defined")
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

// parseStopTimeout ...
func (cfg *rawConfig) parseStopTimeout() (int, error) {
	if cfg.stopTimeout == 0 {
		return defaultStopTimeout, nil
	}
	return cfg.stopTimeout, nil
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
			return fmt.Errorf("could not write config file: %s", err)
		}
	} else {
		return fmt.Errorf("-render flag is invalid: '%s'", renderFlag)
	}

	return nil
}

// LoadConfig parses and validates the raw config values
func LoadConfig(configFlag string) (*Config, error) {

	template, err := renderConfigTemplate(configFlag)
	if err != nil {
		return nil, err
	}
	configMap, err := unmarshalConfig(template)
	if err != nil {
		return nil, err
	}
	disc, err := parseDiscoveryBackend(configMap)
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
	cfg.Discovery = disc

	cfg.LogConfig = raw.logConfig

	stopTimeout, err := raw.parseStopTimeout()
	if err != nil {
		return nil, err
	}
	cfg.StopTimeout = stopTimeout

	serviceConfigs, err := services.NewConfigs(raw.services, disc)
	if err != nil {
		return nil, fmt.Errorf("unable to parse services: %v", err)
	}
	cfg.Services = serviceConfigs

	checks, err := checks.NewConfigs(raw.services)
	if err != nil {
		return nil, fmt.Errorf("unable to parse checks: %v", err)
	}
	cfg.Checks = checks

	watches, err := watches.NewConfigs(raw.watches, disc)
	if err != nil {
		return nil, fmt.Errorf("unable to parse watches: %v", err)
	}
	cfg.Watches = watches

	telemetry, err := telemetry.NewConfig(raw.telemetry, disc)
	if err != nil {
		return nil, err
	}
	if telemetry != nil {
		cfg.Telemetry = telemetry
		cfg.Services = append(cfg.Services, telemetry.ServiceConfig)
	}

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
			return nil, fmt.Errorf("could not read config file: %s", err)
		}
	} else {
		data = []byte(configFlag)
	}
	template, err := ApplyTemplate(data)
	if err != nil {
		err = fmt.Errorf("could not apply template to config: %v", err)
	}
	return template, err
}

func unmarshalConfig(data []byte) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		syntax, ok := err.(*json.SyntaxError)
		if !ok {
			return nil, fmt.Errorf(
				"could not parse configuration: %s",
				err)
		}
		return nil, newJSONparseError(data, syntax)
	}
	return config, nil
}

func newJSONparseError(js []byte, syntax *json.SyntaxError) error {
	line, col, err := highlightError(js, syntax.Offset)
	return fmt.Errorf("parse error at line:col [%d:%d]: %s\n%s", line, col, syntax, err)
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
	result.services = decodeArray(configMap["services"])
	result.watches = decodeArray(configMap["backends"])
	result.telemetry = configMap["telemetry"]

	delete(configMap, "logging")
	delete(configMap, "stopTimeout")
	delete(configMap, "services")
	delete(configMap, "backends")
	delete(configMap, "telemetry")
	var unused []string
	for key := range configMap {
		unused = append(unused, key)
	}
	if len(unused) > 0 {
		return fmt.Errorf("unknown config keys: %v", unused)
	}
	return nil
}
