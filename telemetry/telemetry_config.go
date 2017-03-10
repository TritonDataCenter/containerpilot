package telemetry

import (
	"fmt"
	"net"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/utils"
)

// TelemetryConfig represents the service to advertise for finding the metrics
// endpoint, and the collection of Sensors.
type TelemetryConfig struct {
	Port       int           `mapstructure:"port"`
	Interfaces []interface{} `mapstructure:"interfaces"` // optional override
	Tags       []string      `mapstructure:"tags"`
	Sensors    []interface{} `mapstructure:"sensors"`

	// derived in Validate
	SensorConfigs []*SensorConfig
	ServiceConfig *services.ServiceConfig
	addr          net.TCPAddr
}

// NewTelemetryConfig parses json config into a validated TelemetryConfig
// including a validated ServiceConfig and validated SensorConfigs
func NewTelemetryConfig(raw interface{}, disc discovery.Backend) (*TelemetryConfig, error) {
	cfg := &TelemetryConfig{Port: 9090} // default values
	if err := utils.DecodeRaw(raw, cfg); err != nil {
		return nil, fmt.Errorf("telemetry configuration error: %v", err)
	}
	if err := cfg.Validate(disc); err != nil {
		return nil, fmt.Errorf("telemetry validation error: %v", err)
	}
	if cfg.Sensors != nil {
		// note that we don't return an error if there are no sensors
		// because the prometheus handler will still pick up metrics
		// internal to ContainerPilot (i.e. the golang runtime)
		sensors, err := NewSensorConfigs(cfg.Sensors)
		if err != nil {
			return nil, err
		}
		cfg.SensorConfigs = sensors
	}
	return cfg, nil
}

// Validate ...
func (cfg *TelemetryConfig) Validate(disc discovery.Backend) error {
	ipAddress, err := utils.IPFromInterfaces(cfg.Interfaces)
	if err != nil {
		return err
	}
	ip := net.ParseIP(ipAddress)
	cfg.addr = net.TCPAddr{IP: ip, Port: cfg.Port}
	serviceCfg := cfg.ToServiceConfig()
	if err := serviceCfg.Validate(disc); err != nil {
		return fmt.Errorf("could not validate telemetry service: %v", err)
	}
	cfg.ServiceConfig = serviceCfg
	return nil
}

// ToServiceConfig
func (cfg *TelemetryConfig) ToServiceConfig() *services.ServiceConfig {
	service := &services.ServiceConfig{
		Name:       "containerpilot", // TODO: hard-coded?
		TTL:        15,               // TODO: hard-coded?
		Heartbeat:  5,                // TODO hard-coded?
		Interfaces: cfg.Interfaces,
		Port:       cfg.Port,
		Tags:       cfg.Tags,
	}
	return service
}
