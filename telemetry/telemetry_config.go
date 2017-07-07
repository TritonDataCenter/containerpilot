package telemetry

import (
	"fmt"
	"net"

	"github.com/joyent/containerpilot/config/decoding"
	"github.com/joyent/containerpilot/config/services"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/jobs"
)

// Config represents the service to advertise for finding the metrics
// endpoint, and the collection of Metrics.
type Config struct {
	Port       int           `mapstructure:"port"`
	Interfaces []interface{} `mapstructure:"interfaces"` // optional override
	Tags       []string      `mapstructure:"tags"`
	Metrics    []interface{} `mapstructure:"metrics"`

	// derived in Validate
	MetricConfigs []*MetricConfig
	JobConfig     *jobs.Config
	addr          net.TCPAddr
}

// NewConfig parses json config into a validated Config
// including a validated Config and validated MetricConfigs
func NewConfig(raw interface{}, disc discovery.Backend) (*Config, error) {
	if raw == nil {
		return nil, nil
	}
	cfg := &Config{Port: 9090} // default values
	if err := decoding.ToStruct(raw, cfg); err != nil {
		return nil, fmt.Errorf("telemetry configuration error: %v", err)
	}
	if err := cfg.Validate(disc); err != nil {
		return nil, fmt.Errorf("telemetry validation error: %v", err)
	}
	if cfg.Metrics != nil {
		// note that we don't return an error if there are no metrics
		// because the prometheus handler will still pick up metrics
		// internal to ContainerPilot (i.e. the golang runtime)
		metrics, err := NewMetricConfigs(cfg.Metrics)
		if err != nil {
			return nil, err
		}
		cfg.MetricConfigs = metrics
	}
	return cfg, nil
}

// Validate ...
func (cfg *Config) Validate(disc discovery.Backend) error {
	ipAddress, err := services.IPFromInterfaces(cfg.Interfaces)
	if err != nil {
		return err
	}
	ip := net.ParseIP(ipAddress)
	cfg.addr = net.TCPAddr{IP: ip, Port: cfg.Port}
	jobConfig := cfg.ToJobConfig()
	if err := jobConfig.Validate(disc); err != nil {
		return fmt.Errorf("could not validate telemetry service: %v", err)
	}
	cfg.JobConfig = jobConfig
	return nil
}

// ToJobConfig ...
func (cfg *Config) ToJobConfig() *jobs.Config {
	service := &jobs.Config{
		Name: "containerpilot", // TODO: hard-coded?
		Health: &jobs.HealthConfig{
			TTL:       15, // TODO: hard-coded?
			Heartbeat: 5,  // TODO hard-coded?
		},
		Interfaces: cfg.Interfaces,
		Port:       cfg.Port,
		Tags:       cfg.Tags,
	}
	return service
}
