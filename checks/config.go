package checks

import (
	"fmt"
	"os"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// HealthCheckConfig configures the health check
type HealthCheckConfig struct {
	ID              string
	Name            string `mapstructure:"name"`
	Poll            int    `mapstructure:"poll"` // time in seconds
	pollInterval    time.Duration
	HealthCheckExec interface{} `mapstructure:"health"`
	exec            *commands.Command
	Timeout         string `mapstructure:"timeout"`
	timeout         time.Duration
	definition      *discovery.ServiceDefinition

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

// NewHealthCheckConfigs parses json config into a validated slice of HealthCheckConfigs
func NewHealthCheckConfigs(raw []interface{}) ([]*HealthCheckConfig, error) {
	var checks []*HealthCheckConfig
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
		checks = append(checks, check)
	}
	return checks, nil
}

// Validate ensures HealthCheckConfig meets all requirements
func (check *HealthCheckConfig) Validate() error {
	if err := utils.ValidateServiceName(check.Name); err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	check.ID = fmt.Sprintf("%s-%s", check.Name, hostname)

	if check.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in health check %s", check.Name)
	}
	check.pollInterval = time.Duration(check.Poll) * time.Second
	if check.Timeout == "" {
		check.Timeout = fmt.Sprintf("%ds", check.Poll)
	}
	timeout, err := utils.GetTimeout(check.Timeout)
	if err != nil {
		return err
	}
	check.timeout = timeout

	cmd, err := commands.NewCommand(check.HealthCheckExec, check.timeout)
	if err != nil {
		// TODO: this is config syntax specific and should be updated
		return fmt.Errorf("could not parse `health` in check %s: %s",
			check.Name, err)
	}
	check.exec = cmd

	return nil
}
