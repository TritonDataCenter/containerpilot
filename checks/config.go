package checks

import (
	"fmt"
	"os"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// HealthCheckConfig configures the health check
type HealthCheckConfig struct {
	ID              string
	Name            string      `mapstructure:"name"`
	Poll            int         `mapstructure:"poll"` // time in seconds
	HealthCheckExec interface{} `mapstructure:"health"`
	Timeout         string      `mapstructure:"timeout"`
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

// NewHealthChecks new checks from a raw config
func NewHealthChecks(raw []interface{}) ([]*HealthCheck, error) {
	if raw == nil {
		return []*HealthCheck{}, nil
	}

	var checkcfgs []*HealthCheckConfig
	var checks []*HealthCheck
	if err := utils.DecodeRaw(raw, &checkcfgs); err != nil {
		return nil, fmt.Errorf("HealthCheck configuration error: %v", err)
	}

	for _, checkcfg := range checkcfgs {
		if err := utils.ValidateServiceName(checkcfg.Name); err != nil {
			return nil, err
		}
		hostname, _ := os.Hostname()
		checkcfg.ID = fmt.Sprintf("%s-%s", checkcfg.Name, hostname)

		if checkcfg.Poll < 1 {
			return []*HealthCheck{},
				fmt.Errorf("`poll` must be > 0 in service %s", checkcfg.Name)
		}
		if checkcfg.Timeout == "" {
			checkcfg.Timeout = fmt.Sprintf("%ds", checkcfg.Poll)
		}
		check, err := NewHealthCheck(checkcfg)
		if err != nil {
			return []*HealthCheck{}, err
		}
		checks = append(checks, check)
	}
	return checks, nil
}
