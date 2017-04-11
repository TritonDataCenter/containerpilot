package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/flynn/json5"

	"github.com/joyent/containerpilot/control"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/jobs"
	"github.com/joyent/containerpilot/telemetry"
	"github.com/joyent/containerpilot/utils"
	"github.com/joyent/containerpilot/watches"
)

type rawConfig struct {
	logConfig   *LogConfig
	stopTimeout int
	jobs        []interface{}
	watches     []interface{}
	telemetry   interface{}
	control     interface{}
}

// Config contains the parsed config elements
type Config struct {
	Discovery   discovery.Backend
	LogConfig   *LogConfig
	StopTimeout int
	Jobs        []*jobs.Config
	Watches     []*watches.Config
	Telemetry   *telemetry.Config
	Control     *control.Config
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

// parseStopTimeout makes sure we have a safe default
func (cfg *rawConfig) parseStopTimeout() (int, error) {
	if cfg.stopTimeout == 0 {
		return defaultStopTimeout, nil
	}
	return cfg.stopTimeout, nil
}

// RenderConfig renders the templated config in configFlag to renderFlag.
func RenderConfig(configFlag, renderFlag string) error {
	configData, err := loadConfigFile(configFlag)
	if err != nil {
		return err
	}
	renderedConfig, err := renderConfigTemplate(configData)
	if err != nil {
		return err
	}

	// Save the rendered template, either to stdout or to file
	if renderFlag == "-" {
		fmt.Printf("%s", renderedConfig)
	} else {
		var err error
		if err = ioutil.WriteFile(renderFlag, renderedConfig, 0644); err != nil {
			return fmt.Errorf("could not write config file: %s", err)
		}
	}

	return nil
}

// LoadConfig loads, parses, and validates the configuration
func LoadConfig(configFlag string) (*Config, error) {
	configData, err := loadConfigFile(configFlag)
	if err != nil {
		return nil, err
	}
	renderedConfig, err := renderConfigTemplate(configData)
	if err != nil {
		return nil, err
	}
	config, err := newConfig(renderedConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func loadConfigFile(configFlag string) ([]byte, error) {
	if configFlag == "" {
		return nil, errors.New("-config flag is required")
	}
	data, err := ioutil.ReadFile(configFlag)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %s", err)
	}
	return data, nil
}

func renderConfigTemplate(configData []byte) ([]byte, error) {
	template, err := ApplyTemplate(configData)
	if err != nil {
		err = fmt.Errorf("could not apply template to config: %v", err)
	}
	return template, err
}

// newConfig unmarshals the textual configuration data into the
// validated Config struct that we'll use the run the application
func newConfig(configData []byte) (*Config, error) {
	configMap, err := unmarshalConfig(configData)
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

	controlConfig, err := control.NewConfig(raw.control)
	if err != nil {
		return nil, fmt.Errorf("unable to parse control: %v", err)
	}
	cfg.Control = controlConfig

	jobConfigs, err := jobs.NewConfigs(raw.jobs, disc)
	if err != nil {
		return nil, fmt.Errorf("unable to parse jobs: %v", err)
	}
	cfg.Jobs = jobConfigs

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
		cfg.Jobs = append(cfg.Jobs, telemetry.JobConfig)
	}

	return cfg, nil
}

func unmarshalConfig(data []byte) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json5.Unmarshal(data, &config); err != nil {
		syntax, ok := err.(*json5.SyntaxError)
		if !ok {
			return nil, fmt.Errorf(
				"could not parse configuration: %s",
				err)
		}
		return nil, newJSONparseError(data, syntax)
	}
	return config, nil
}

func newJSONparseError(js []byte, syntax *json5.SyntaxError) error {
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
	result.control = configMap["control"]
	result.jobs = decodeArray(configMap["jobs"])
	result.watches = decodeArray(configMap["watches"])
	result.telemetry = configMap["telemetry"]

	delete(configMap, "logging")
	delete(configMap, "control")
	delete(configMap, "stopTimeout")
	delete(configMap, "jobs")
	delete(configMap, "watches")
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
