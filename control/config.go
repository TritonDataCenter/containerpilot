package control

import (
	"fmt"

	"github.com/joyent/containerpilot/utils"
)

const (
	// DefaultSocket is the default location of the unix domain socket file
	DefaultSocket = "/var/run/containerpilot.socket"
)

// Config represents the location on the file system which serves the Unix
// control socket file.
type Config struct {
	SocketPath string `mapstructure:"socket"`
}

// NewConfig parses a json config into a validated Config used by control
// Server.
func NewConfig(raw interface{}) (*Config, error) {
	cfg := &Config{SocketPath: DefaultSocket} // defaults
	if raw == nil {
		return cfg, nil
	}

	if err := utils.DecodeRaw(raw, cfg); err != nil {
		return nil, fmt.Errorf("control config parsing error: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("control config validation error: %v", err)
	}

	return cfg, nil
}

// Validate parsed control configuration and the values contained within.
func (cfg *Config) Validate() error {
	// TODO: Validate NestedConfig and socket's file system location ...
	return nil
}
