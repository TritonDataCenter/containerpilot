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
		if err := utils.ValidateServiceName(watchcfg.Name); err != nil {
			return nil, err
		}
		if watchcfg.OnChangeExec == nil {
			return nil, fmt.Errorf("`onChange` is required in watch %s",
				watchcfg.Name)
		}
		if watchcfg.Timeout == "" {
			watchcfg.Timeout = fmt.Sprintf("%ds", watchcfg.Poll)
		}
		if watchcfg.Poll < 1 {
			return nil, fmt.Errorf("`poll` must be > 0 in watch %s",
				watchcfg.Name)
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
