// Package config is the top-level configuration parsing package
package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/flynn/json5"

	"github.com/tritondatacenter/containerpilot/config/decode"
	"github.com/tritondatacenter/containerpilot/config/logger"
	"github.com/tritondatacenter/containerpilot/config/template"
	"github.com/tritondatacenter/containerpilot/control"
	"github.com/tritondatacenter/containerpilot/discovery"
	"github.com/tritondatacenter/containerpilot/jobs"
	"github.com/tritondatacenter/containerpilot/telemetry"
	"github.com/tritondatacenter/containerpilot/watches"
)

type rawConfig struct {
	consul      interface{}
	logConfig   *logger.Config
	stopTimeout int
	jobs        []interface{}
	watches     []interface{}
	telemetry   interface{}
	control     interface{}
}

// Config contains the parsed config elements
type Config struct {
	Discovery   discovery.Backend
	LogConfig   *logger.Config
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

// InitLogging configure logrus with the new log config if available
func (cfg *Config) InitLogging() error {
	if cfg.LogConfig != nil {
		return cfg.LogConfig.Init()
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
	if renderFlag == "-" || renderFlag == "" {
		fmt.Printf("%s", renderedConfig)
	} else {
		var err error
		if err = os.WriteFile(renderFlag, renderedConfig, 0644); err != nil {
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
	data, err := os.ReadFile(configFlag)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %s", err)
	}
	return data, nil
}

func renderConfigTemplate(configData []byte) ([]byte, error) {
	templ, err := template.Apply(configData)
	if err != nil {
		err = fmt.Errorf("could not apply template to config: %v", err)
	}
	return templ, err
}

// newConfig unmarshals the textual configuration data into the
// validated Config struct that we'll use the run the application
func newConfig(configData []byte) (*Config, error) {
	configMap, err := unmarshalConfig(configData)
	if err != nil {
		return nil, err
	}

	raw := &rawConfig{}
	if err = decodeConfig(configMap, raw); err != nil {
		return nil, err
	}
	cfg := &Config{}

	disc, err := discovery.NewConsul(raw.consul)
	if err != nil {
		return nil, err
	}
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

// We can't use mapstructure to decode our config map since we want the values
// to also be raw interface{} types. mapstructure can only decode
// into concrete structs and primitives
func decodeConfig(configMap map[string]interface{}, result *rawConfig) error {
	var logConfig logger.Config
	var stopTimeout int
	if err := decode.ToStruct(configMap["logging"], &logConfig); err != nil {
		return err
	}
	if err := decode.ToStruct(configMap["stopTimeout"], &stopTimeout); err != nil {
		return err
	}
	result.consul = configMap["consul"]
	result.stopTimeout = stopTimeout
	result.logConfig = &logConfig
	result.control = configMap["control"]
	result.jobs = decode.ToSlice(configMap["jobs"])
	result.watches = decode.ToSlice(configMap["watches"])
	result.telemetry = configMap["telemetry"]

	delete(configMap, "consul")
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
