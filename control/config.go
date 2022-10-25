package control

import (
	"fmt"

	"github.com/tritondatacenter/containerpilot/config/decode"
)

// DefaultSocket is the default location of the unix domain socket file
var DefaultSocket = "/var/run/containerpilot.socket"

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

	if err := decode.ToStruct(raw, cfg); err != nil {
		return nil, fmt.Errorf("control config parsing error: %v", err)
	}

	return cfg, nil
}
