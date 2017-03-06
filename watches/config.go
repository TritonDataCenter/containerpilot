package watches

import (
	"fmt"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

type WatchConfig struct {
	Name             string      `mapstructure:"name"`
	Poll             int         `mapstructure:"poll"` // time in seconds
	OnChangeExec     interface{} `mapstructure:"onChange"`
	Tag              string      `mapstructure:"tag"`
	Timeout          string      `mapstructure:"timeout"`
	discoveryService discovery.ServiceBackend
}

// NewWatches creates a new watch from a raw config structure
func NewWatches(raw []interface{}, disc discovery.ServiceBackend) ([]*Watch, error) {
	if raw == nil {
		return []*Watch{}, nil
	}
	var watchcfgs []*WatchConfig
	var watches []*Watch
	if err := utils.DecodeRaw(raw, &watchcfgs); err != nil {
		return nil, fmt.Errorf("Watch configuration error: %v", err)
	}
	for _, watchcfg := range watchcfgs {
		if err := validateWatchConfig(watchcfg); err != nil {
			return []*Watch{}, err
		}
		watchcfg.discoveryService = disc
		watch, err := NewWatch(watchcfg)
		if err != nil {
			return []*Watch{}, err
		}
		watches = append(watches, watch)
	}
	return watches, nil
}

// ensure WatchConfig meets all requirements
func validateWatchConfig(cfg *WatchConfig) error {
	if err := utils.ValidateServiceName(cfg.Name); err != nil {
		return err
	}
	if cfg.OnChangeExec == nil {
		return fmt.Errorf("`onChange` is required in watch %s", cfg.Name)
	}
	if cfg.Timeout == "" {
		cfg.Timeout = fmt.Sprintf("%ds", cfg.Poll)
	}
	if cfg.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in watch %s", cfg.Name)
	}
	return nil
}
