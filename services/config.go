package services

import (
	"fmt"
	"os"
	"time"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// ServiceConfig holds the configuration for service discovery data
type ServiceConfig struct {
	ID               string      // used only for ServiceDefinition
	Exec             interface{} // TODO: this will be parsed from config when we update syntax
	Name             string      `mapstructure:"name"`
	Heartbeat        int         `mapstructure:"poll"` // time in seconds
	Port             int         `mapstructure:"port"`
	TTL              int         `mapstructure:"ttl"`
	Interfaces       interface{} `mapstructure:"interfaces"`
	Tags             []string    `mapstructure:"tags"`
	IPAddress        string
	ConsulConfig     *ConsulConfig `mapstructure:"consul"`
	discoveryService discovery.ServiceBackend
	definition       *discovery.ServiceDefinition
	execTimeout      string // TODO: this will be parsed from config when we update syntax

	/* TODO:
	These fields are here *only* so we can reuse the config map we use
	in the checks package here too. this package ignores them. when we
	move on to the v3 configuration syntax these will be dropped.
	*/
	checkExec    interface{} `mapstructure:"health"`
	checkTimeout string      `mapstructure:"timeout"`
}

// ConsulConfig handles additional Consul configuration.
type ConsulConfig struct {
	EnableTagOverride              bool   `mapstructure:"enableTagOverride"`
	DeregisterCriticalServiceAfter string `mapstructure:"deregisterCriticalServiceAfter"`
}

// NewServices new services from a raw config
func NewServices(raw []interface{}, disc discovery.ServiceBackend) ([]*Service, error) {
	if raw == nil {
		return []*Service{}, nil
	}
	var (
		services    []*Service
		servicecfgs []*ServiceConfig
	)
	if err := utils.DecodeRaw(raw, &servicecfgs); err != nil {
		return nil, fmt.Errorf("service configuration error: %v", err)
	}
	for _, servicecfg := range servicecfgs {
		if err := validateServiceConfig(servicecfg, disc); err != nil {
			return nil, err
		}
		service, err := NewService(servicecfg)
		if err != nil {
			return []*Service{}, err
		}
		services = append(services, service)
	}
	return services, nil
}

// TODO: we should further break up the config parsing of the service
// from the config for the Consul service definition
func (cfg *ServiceConfig) AddDiscoveryConfig(disc discovery.ServiceBackend) error {
	cfg.discoveryService = disc
	var consulExtras *discovery.ConsulExtras
	if cfg.ConsulConfig != nil {
		if cfg.ConsulConfig.DeregisterCriticalServiceAfter != "" {
			if _, err := time.ParseDuration(cfg.ConsulConfig.DeregisterCriticalServiceAfter); err != nil {
				return fmt.Errorf("could not parse consul `deregisterCriticalServiceAfter` in service %s: %s", cfg.Name, err)
			}
		}
		consulExtras = &discovery.ConsulExtras{
			DeregisterCriticalServiceAfter: cfg.ConsulConfig.DeregisterCriticalServiceAfter,
			EnableTagOverride:              cfg.ConsulConfig.EnableTagOverride,
		}
	}
	cfg.definition = &discovery.ServiceDefinition{
		ID:           cfg.ID,
		Name:         cfg.Name,
		Port:         cfg.Port,
		TTL:          cfg.TTL,
		Tags:         cfg.Tags,
		IPAddress:    cfg.IPAddress,
		ConsulExtras: consulExtras,
	}
	return nil
}

func validateServiceConfig(cfg *ServiceConfig, disc discovery.ServiceBackend) error {
	if err := utils.ValidateServiceName(cfg.Name); err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	cfg.ID = fmt.Sprintf("%s-%s", cfg.Name, hostname)

	// if port isn't set then we won't do any discovery for this service
	if cfg.Port == 0 {
		if cfg.Heartbeat > 0 || cfg.TTL > 0 {
			return fmt.Errorf("`heartbeat` and `ttl` may not be set in service `%s` if `port` is not set", cfg.Name)
		}
		return nil
	}
	if cfg.Heartbeat < 1 {
		return fmt.Errorf("`poll` must be > 0 in service `%s` when `port` is set", cfg.Name)
	}
	if cfg.TTL < 1 {
		return fmt.Errorf("`ttl` must be > 0 in service `%s` when `port` is set", cfg.Name)
	}

	interfaces, ifaceErr := utils.ToStringArray(cfg.Interfaces)
	if ifaceErr != nil {
		return ifaceErr
	}

	ipAddress, err := utils.GetIP(interfaces)
	if err != nil {
		return err
	}
	cfg.IPAddress = ipAddress

	if err := cfg.AddDiscoveryConfig(disc); err != nil {
		return err
	}

	return nil
}
