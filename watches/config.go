package watches

import (
	"fmt"

	"github.com/tritondatacenter/containerpilot/config/decode"
	"github.com/tritondatacenter/containerpilot/config/services"
	"github.com/tritondatacenter/containerpilot/discovery"
)

// Config configures the watch
type Config struct {
	Name             string `mapstructure:"name"`
	serviceName      string
	Poll             int    `mapstructure:"interval"` // time in seconds
	Tag              string `mapstructure:"tag"`
	DC               string `mapstructure:"dc"` // Consul datacenter
	discoveryService discovery.Backend
}

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}, disc discovery.Backend) ([]*Config, error) {
	var watches []*Config
	if raw == nil {
		return watches, nil
	}
	if err := decode.ToStruct(raw, &watches); err != nil {
		return watches, fmt.Errorf("Watch configuration error: %v", err)
	}
	for _, watch := range watches {
		if err := watch.Validate(disc); err != nil {
			return watches, err
		}
	}
	return watches, nil
}

// Validate ensures Config meets all requirements
func (cfg *Config) Validate(disc discovery.Backend) error {
	if err := services.ValidateName(cfg.Name); err != nil {
		return err
	}

	cfg.serviceName = cfg.Name
	cfg.Name = "watch." + cfg.Name

	if cfg.Poll < 1 {
		return fmt.Errorf("watch[%s].interval must be > 0", cfg.serviceName)
	}
	cfg.discoveryService = disc
	return nil
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (cfg *Config) String() string {
	return "watches.Config[" + cfg.Name + "]"
}
