package watches

import (
	"fmt"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// Config configures the watch
type Config struct {
	Name             string `mapstructure:"name"`
	serviceName      string
	Poll             int    `mapstructure:"poll"` // time in seconds
	Tag              string `mapstructure:"tag"`
	discoveryService discovery.Backend
}

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}, disc discovery.Backend) ([]*Config, error) {
	var watches []*Config
	if raw == nil {
		return watches, nil
	}
	if err := utils.DecodeRaw(raw, &watches); err != nil {
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
	if err := utils.ValidateServiceName(cfg.Name); err != nil {
		return err
	}

	cfg.serviceName = cfg.Name
	cfg.Name = "watch." + cfg.Name

	if cfg.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in watch %s", cfg.serviceName)
	}
	cfg.discoveryService = disc
	return nil
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (cfg *Config) String() string {
	return "watches.Config[" + cfg.Name + "]"
}
