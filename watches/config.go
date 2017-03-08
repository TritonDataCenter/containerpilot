package watches

import (
	"fmt"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

type WatchConfig struct {
	Name             string      `mapstructure:"name"`
	Poll             int         `mapstructure:"poll"` // time in seconds
	OnChangeExec     interface{} `mapstructure:"onChange"`
	onChangeExec     *commands.Command
	Tag              string `mapstructure:"tag"`
	Timeout          string `mapstructure:"timeout"`
	timeout          time.Duration
	discoveryService discovery.Backend
}

// NewWatchConfigs parses json config into a validated slice of WatchConfigs
func NewWatchConfigs(raw []interface{}, disc discovery.Backend) ([]*WatchConfig, error) {
	var watches []*WatchConfig
	if raw == nil {
		return watches, nil
	}
	if err := utils.DecodeRaw(raw, &watches); err != nil {
		return watches, fmt.Errorf("Watch configuration error: %v", err)
	}
	for _, watch := range watches {
		if err := watch.Validate(); err != nil {
			return watches, err
		}
		watch.discoveryService = disc
		watches = append(watches, watch)
	}
	return watches, nil
}

// Validate ensures WatchConfig meets all requirements
func (cfg *WatchConfig) Validate() error {
	if err := utils.ValidateServiceName(cfg.Name); err != nil {
		return err
	}
	if cfg.OnChangeExec == nil {
		// TODO: this error message is tied to existing config syntax
		return fmt.Errorf("`onChange` is required in watch %s", cfg.Name)
	}
	if cfg.Timeout == "" {
		cfg.Timeout = fmt.Sprintf("%ds", cfg.Poll)
	}
	timeout, err := utils.GetTimeout(cfg.Timeout)
	if err != nil {
		return err
	}
	cfg.timeout = timeout

	if cfg.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in watch %s", cfg.Name)
	}
	cmd, err := commands.NewCommand(cfg.OnChangeExec, cfg.timeout)
	if err != nil {
		// TODO: this error message is tied to existing config syntax
		return fmt.Errorf("could not parse `onChange` in watch %s: %s",
			cfg.Name, err)
	}
	cfg.onChangeExec = cmd

	return nil
}
