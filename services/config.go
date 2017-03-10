package services

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/utils"
)

const taskMinDuration = time.Millisecond

// ServiceConfig holds the configuration for service discovery data
type ServiceConfig struct {
	ID                string      // used only for ServiceDefinition
	Exec              interface{} // TODO: this will be parsed from config when we update syntax
	Name              string      `mapstructure:"name"`
	Heartbeat         int         `mapstructure:"poll"` // time in seconds
	heartbeatInterval time.Duration
	Port              int         `mapstructure:"port"`
	TTL               int         `mapstructure:"ttl"`
	Interfaces        interface{} `mapstructure:"interfaces"`
	Tags              []string    `mapstructure:"tags"`
	ipAddress         string
	ConsulConfig      *ConsulConfig `mapstructure:"consul"`
	discoveryService  discovery.Backend
	definition        *discovery.ServiceDefinition
	exec              *commands.Command
	ExecTimeout       string // TODO: this will be parsed from config when we update syntax
	execTimeout       time.Duration

	// TODO: currently this will only appear when we create a ServiceConfig
	// from a CoprocessConfig or TaskConfig
	Restarts     interface{} `mapstructure:"restarts"`
	Frequency    string      `mapstructure:"frequency"`
	restart      bool
	restartLimit int
	freqInterval time.Duration

	startupEvent   events.Event
	startupTimeout time.Duration

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

// NewServiceConfigs parses json config into a validated slice of ServiceConfigs
func NewServiceConfigs(raw []interface{}, disc discovery.Backend) ([]*ServiceConfig, error) {
	var services []*ServiceConfig
	if raw == nil {
		return services, nil
	}
	if err := utils.DecodeRaw(raw, &services); err != nil {
		return nil, fmt.Errorf("service configuration error: %v", err)
	}
	for _, service := range services {
		if err := service.Validate(disc); err != nil {
			return nil, err
		}
	}
	return services, nil
}

// Validate ensures that a ServiceConfig meets all constraints
func (cfg *ServiceConfig) Validate(disc discovery.Backend) error {
	if disc != nil {
		// non-advertised services don't need to have their names validated
		if err := utils.ValidateServiceName(cfg.Name); err != nil {
			return err
		}
	}
	hostname, _ := os.Hostname()
	cfg.ID = fmt.Sprintf("%s-%s", cfg.Name, hostname)

	// if port isn't set then we won't do any discovery for this service
	if cfg.Port == 0 {
		if cfg.Heartbeat > 0 || cfg.TTL > 0 {
			return fmt.Errorf("`heartbeat` and `ttl` may not be set in service `%s` if `port` is not set", cfg.Name)
		}
	} else {
		if cfg.Heartbeat < 1 {
			return fmt.Errorf("`poll` must be > 0 in service `%s` when `port` is set", cfg.Name)
		}
		if cfg.TTL < 1 {
			return fmt.Errorf("`ttl` must be > 0 in service `%s` when `port` is set", cfg.Name)
		}
	}

	cfg.heartbeatInterval = time.Duration(cfg.Heartbeat) * time.Second
	if err := configureFrequency(cfg); err != nil {
		return err
	}
	//	cfg.startupTimeout = 0 // TODO: need to expose this as a config value
	// cfg.startupEvent = events.GlobalStartup// TODO: need to expose this as a config value

	if err := configureRestarts(cfg); err != nil {
		return err
	}

	if cfg.ExecTimeout != "" {
		execTimeout, err := utils.GetTimeout(cfg.ExecTimeout)
		if err != nil {
			return fmt.Errorf("could not parse `timeout` for service %s: %v", cfg.Name, err)
		}
		cfg.execTimeout = execTimeout
	}
	if cfg.Exec != nil {
		cmd, err := commands.NewCommand(cfg.Exec, cfg.execTimeout)
		if err != nil {
			return fmt.Errorf("could not parse `exec` for service %s: %s", cfg.Name, err)
		}
		cmd.Name = cfg.Name
		cfg.exec = cmd
	}

	interfaces, ifaceErr := utils.ToStringArray(cfg.Interfaces)
	if ifaceErr != nil {
		return ifaceErr
	}

	ipAddress, err := utils.GetIP(interfaces)
	if err != nil {
		return err
	}
	cfg.ipAddress = ipAddress

	if err := cfg.AddDiscoveryConfig(disc); err != nil {
		return err
	}

	return nil
}

func configureFrequency(cfg *ServiceConfig) error {
	if cfg.Frequency == "" {
		// defaults if omitted
		return nil
	}
	freq, err := utils.ParseDuration(cfg.Frequency)
	if err != nil {
		return fmt.Errorf("unable to parse frequency '%s': %v", cfg.Frequency, err)
	}
	if freq < taskMinDuration {
		return fmt.Errorf("frequency '%s' cannot be less than %v", cfg.Frequency, taskMinDuration)
	}
	cfg.freqInterval = freq
	return nil
}

func configureRestarts(cfg *ServiceConfig) error {

	// defaults if omitted
	if cfg.Restarts == nil {
		cfg.restart = false
		cfg.restartLimit = 0
		return nil
	}

	const msg = `invalid 'restarts' field "%v": accepts positive integers, "unlimited" or "never"`

	switch t := cfg.Restarts.(type) {
	case string:
		if t == "unlimited" {
			cfg.restartLimit = unlimitedRestarts
		} else if t == "never" {
			cfg.restartLimit = 0
		} else if i, err := strconv.Atoi(t); err == nil && i >= 0 {
			cfg.restartLimit = i
		} else {
			return fmt.Errorf(msg, cfg.Restarts)
		}
	case float64, int:
		// mapstructure can figure out how to decode strings into int fields
		// but doesn't try to guess and just gives us a float64 if it's got
		// a number that it's decoding to an interface{}. We'll only return
		// an error if we can't cast it to an int. This means that an end-user
		// can pass in `restarts: 1.2` and have undocumented truncation but the
		// wtf would be all on them at that point.
		if i, ok := t.(int); ok && i >= 0 {
			cfg.restartLimit = i
		} else if i, ok := t.(float64); ok && i >= 0 {
			cfg.restartLimit = int(i)
		} else {
			return fmt.Errorf(msg, cfg.Restarts)
		}
	default:
		return fmt.Errorf(msg, cfg.Restarts)
	}

	cfg.restart = (cfg.restartLimit > 0 ||
		cfg.restartLimit == unlimitedRestarts)

	return nil
}

// AddDiscoveryConfig ...
func (cfg *ServiceConfig) AddDiscoveryConfig(disc discovery.Backend) error {
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
		IPAddress:    cfg.ipAddress,
		ConsulExtras: consulExtras,
	}
	return nil
}
