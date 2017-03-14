package checks

import (
	"fmt"
	"os"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/utils"
)

// Config configures the health check
type Config struct {
	ID           string
	Name         string `mapstructure:"name"`
	Poll         int    `mapstructure:"poll"` // time in seconds
	pollInterval time.Duration
	Exec         interface{} `mapstructure:"health"`
	exec         *commands.Command
	Timeout      string `mapstructure:"timeout"`
	timeout      time.Duration

	/* TODO:
	These fields are here *only* so we can reuse the config map we use
	in the services package here too. this package ignores them. when we
	move on to the v3 configuration syntax these will be dropped.
	*/
	serviceTTL        int         `mapstructure:"ttl"`
	serviceInterfaces interface{} `mapstructure:"interfaces"`
	serviceTags       []string    `mapstructure:"tags"`
	servicePort       int         `mapstructure:"port"`
}

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}) ([]*Config, error) {
	var checks []*Config
	if raw == nil {
		return checks, nil
	}
	if err := utils.DecodeRaw(raw, &checks); err != nil {
		return nil, fmt.Errorf("HealthCheck configuration error: %v", err)
	}
	for _, check := range checks {
		err := check.Validate()
		if err != nil {
			return checks, err
		}
	}
	return checks, nil
}

// Validate ensures Config meets all requirements
func (cfg *Config) Validate() error {
	if err := utils.ValidateServiceName(cfg.Name); err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	cfg.ID = fmt.Sprintf("%s-%s", cfg.Name, hostname)

	if cfg.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in health check %s", cfg.Name)
	}
	cfg.pollInterval = time.Duration(cfg.Poll) * time.Second
	if cfg.Timeout == "" {
		cfg.Timeout = fmt.Sprintf("%ds", cfg.Poll)
	}
	timeout, err := utils.GetTimeout(cfg.Timeout)
	if err != nil {
		return fmt.Errorf("could not parse `timeout` in check %s: %v", cfg.Name, err)
	}
	cfg.timeout = timeout

	cmd, err := commands.NewCommand(cfg.Exec, cfg.timeout)
	if err != nil {
		// TODO: this is config syntax specific and should be updated
		return fmt.Errorf("could not parse `health` in check %s: %s",
			cfg.Name, err)
	}
	cmd.Name = cfg.Name
	cfg.exec = cmd

	return nil
}
