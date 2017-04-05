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
	Name         string `mapstructure:"name"`
	Poll         int    `mapstructure:"poll"` // time in seconds
	pollInterval time.Duration
	Exec         interface{} `mapstructure:"health"`
	exec         *commands.Command
	Timeout      string `mapstructure:"timeout"`
	timeout      time.Duration
	jobName      string // for now always the same as the Name

	/* TODO v3:
	These fields are here *only* so we can reuse the config map we use
	in the services package here too. this package ignores them. when we
	move on to the v3 configuration syntax these will be dropped.
	*/
	serviceTTL         int         `mapstructure:"ttl"`
	serviceInterfaces  interface{} `mapstructure:"interfaces"`
	serviceTags        []string    `mapstructure:"tags"`
	servicePort        int         `mapstructure:"port"`
	serviceExec        interface{} `mapstructure:"exec"`
	serviceExecTimeout interface{} `mapstructure:"execTimeout"`
	servicePreStart    interface{} `mapstructure:"preStart"`
	servicePreStop     interface{} `mapstructure:"preStop"`
	servicePostStop    interface{} `mapstructure:"postStop"`
	serviceRestarts    interface{} `mapstructure:"restarts"`
	serviceFrequency   interface{} `mapstructure:"frequency"`
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
		// TODO v3: we'll remove this check when we split the check
		// from the service config
		if check.Exec != nil {
			err := check.Validate()
			if err != nil {
				return validatedChecks, err
			}
			validatedChecks = append(validatedChecks, check)
		}
	}
	return validatedChecks, nil
}

// Validate ensures Config meets all requirements
func (cfg *Config) Validate() error {
	if err := utils.ValidateServiceName(cfg.Name); err != nil {
		return err
	}
	cfg.jobName = cfg.Name
	cfg.Name = cfg.Name + ".check"

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

	cmd, err := commands.NewCommand(cfg.Exec, cfg.timeout,
		log.Fields{"job": cfg.jobName, "check": cfg.Name})
	if err != nil {
		// TODO v3: this is config syntax specific and should be updated
		return fmt.Errorf("could not parse `health` in check %s: %s",
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
