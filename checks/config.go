package checks

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/utils"
)

// Config configures the health check
type Config struct {
	Name         string      `mapstructure:"name"`
	Job          string      `mapstructure:"job"`
	Poll         int         `mapstructure:"poll"` // time in seconds
	Exec         interface{} `mapstructure:"exec"`
	Timeout      string      `mapstructure:"timeout"`
	pollInterval time.Duration
	exec         *commands.Command
	timeout      time.Duration
}

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}) ([]*Config, error) {
	var (
		unvalidatedChecks []*Config
		validatedChecks   []*Config
	)
	if raw == nil {
		return validatedChecks, nil
	}
	if err := utils.DecodeRaw(raw, &unvalidatedChecks); err != nil {
		return nil, fmt.Errorf("HealthCheck configuration error: %v", err)
	}
	for _, check := range unvalidatedChecks {
		err := check.Validate()
		if err != nil {
			return validatedChecks, err
		}
		validatedChecks = append(validatedChecks, check)
	}
	return validatedChecks, nil
}

// Validate ensures Config meets all requirements
func (cfg *Config) Validate() error {
	if cfg.Job == "" {
		cfg.Job = cfg.Name
	}
	cfg.Name = "check." + cfg.Name
	if err := utils.ValidateServiceName(cfg.Job); err != nil {
		return err
	}

	if cfg.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in health check %s", cfg.Name)
	}
	cfg.pollInterval = time.Duration(cfg.Poll) * time.Second
	if cfg.Timeout == "" {
		cfg.Timeout = fmt.Sprintf("%ds", cfg.Poll)
	}
	timeout, err := utils.GetTimeout(cfg.Timeout)
	if err != nil {
		return fmt.Errorf("could not parse `timeout` in health check %s: %v", cfg.Name, err)
	}
	cfg.timeout = timeout

	cmd, err := commands.NewCommand(cfg.Exec, cfg.timeout,
		log.Fields{"job": cfg.Job, "check": cfg.Name})
	if err != nil {
		return fmt.Errorf("could not parse `exec` in health check %s: %s",
			cfg.Name, err)
	}
	cmd.Name = cfg.Name
	cfg.exec = cmd

	return nil
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (cfg *Config) String() string {
	return "checks.Config[" + cfg.Name + "]"
}
