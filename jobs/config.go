package jobs

import (
	"fmt"
	"os"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/utils"
)

const taskMinDuration = time.Millisecond

// Config holds the configuration for service discovery data
type Config struct {
	Name string      `mapstructure:"name"`
	Exec interface{} `mapstructure:"exec"`

	// heartbeat and service discovery config
	Heartbeat         int           `mapstructure:"heartbeat"` // time in seconds
	TTL               int           `mapstructure:"ttl"`
	Port              int           `mapstructure:"port"`
	Interfaces        interface{}   `mapstructure:"interfaces"`
	Tags              []string      `mapstructure:"tags"`
	ConsulConfig      *ConsulConfig `mapstructure:"consul"`
	heartbeatInterval time.Duration
	discoveryCatalog  discovery.Backend
	definition        *discovery.ServiceDefinition

	// timeouts and restarts
	ExecTimeout     string      `mapstructure:"execTimeout"`
	Restarts        interface{} `mapstructure:"restarts"`
	Frequency       string      `mapstructure:"frequency"`
	StopTimeout     string      `mapstructure:"stopTimeout"`
	execTimeout     time.Duration
	exec            *commands.Command
	stoppingTimeout time.Duration
	restartLimit    int
	freqInterval    time.Duration

	// related jobs
	When              *WhenConfig `mapstructure:"when"`
	whenEvent         events.Event
	whenTimeout       time.Duration
	stoppingWaitEvent events.Event
}

// WhenConfig determines when a Job runs (dependencies on other Jobs
// or on Watches)
type WhenConfig struct {
	Source  string `mapstructure:"source"`
	Event   string `mapstructure:"event"`
	Timeout string `mapstructure:"timeout"`
}

// ConsulConfig handles additional Consul configuration.
type ConsulConfig struct {
	EnableTagOverride              bool   `mapstructure:"enableTagOverride"`
	DeregisterCriticalServiceAfter string `mapstructure:"deregisterCriticalServiceAfter"`
}

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}, disc discovery.Backend) ([]*Config, error) {
	var jobs []*Config
	if raw == nil {
		return jobs, nil
	}
	if err := utils.DecodeRaw(raw, &jobs); err != nil {
		return nil, fmt.Errorf("job configuration error: %v", err)
	}
	stopDependencies := make(map[string]string)
	for _, job := range jobs {
		if err := job.Validate(disc); err != nil {
			return nil, err
		}
		if job.whenEvent.Code == events.Stopping {
			stopDependencies[job.whenEvent.Source] = job.Name
		}
	}
	// set up any dependencies on "stopping" events
	for _, job := range jobs {
		if dependent, ok := stopDependencies[job.Name]; ok {
			job.setStopping(dependent)
		}
	}
	return jobs, nil
}

// Validate ensures that a Config meets all constraints
func (cfg *Config) Validate(disc discovery.Backend) error {
	if disc != nil {
		// non-advertised jobs don't need to have their names validated
		if err := utils.ValidateServiceName(cfg.Name); err != nil {
			return err
		}
	}
	if err := cfg.validateDiscovery(disc); err != nil {
		return err
	}
	if err := cfg.validateFrequency(); err != nil {
		return err
	}
	if err := cfg.validateDependencies(); err != nil {
		return err
	}
	if err := cfg.validateStoppingTimeout(); err != nil {
		return err
	}
	if err := cfg.validateRestarts(); err != nil {
		return err
	}
	if err := cfg.validateExec(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) setStopping(name string) {
	cfg.stoppingWaitEvent = events.Event{events.Stopped, name}
}

func (cfg *Config) validateDiscovery(disc discovery.Backend) error {
	// if port isn't set then we won't do any discovery for this job
	if cfg.Port == 0 {
		if cfg.Heartbeat > 0 || cfg.TTL > 0 {
			return fmt.Errorf("`heartbeat` and `ttl` may not be set in job `%s` if `port` is not set", cfg.Name)
		}
		return nil
	}
	if cfg.Heartbeat < 1 {
		return fmt.Errorf("`heartbeat` must be > 0 in job `%s` when `port` is set", cfg.Name)
	}
	if cfg.TTL < 1 {
		return fmt.Errorf("`ttl` must be > 0 in job `%s` when `port` is set", cfg.Name)
	}
	cfg.heartbeatInterval = time.Duration(cfg.Heartbeat) * time.Second
	if err := cfg.AddDiscoveryConfig(disc); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) validateFrequency() error {
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

func (cfg *Config) validateDependencies() error {
	if cfg.When != nil {
		whenTimeout, err := utils.GetTimeout(cfg.When.Timeout)
		if err != nil {
			return fmt.Errorf("could not parse `when` timeout for job %s: %v",
				cfg.Name, err)
		}
		cfg.whenTimeout = whenTimeout
		eventCode, err := events.FromString(cfg.When.Event)
		if err != nil {
			return fmt.Errorf("could not parse `when` event for job %s: %v",
				cfg.Name, err)
		}
		cfg.whenEvent = events.Event{eventCode, cfg.When.Source}
	} else {
		cfg.whenTimeout = time.Duration(0)
		cfg.whenEvent = events.GlobalStartup
	}
	return nil
}

func (cfg *Config) validateStoppingTimeout() error {
	stoppingTimeout, err := utils.GetTimeout(cfg.StopTimeout)
	if err != nil {
		return fmt.Errorf("could not parse `stopTimeout` for job %s: %v",
			cfg.Name, err)
	}
	cfg.stoppingTimeout = stoppingTimeout
	cfg.stoppingWaitEvent = events.NonEvent
	return nil
}

func (cfg *Config) validateExec() error {

	if cfg.ExecTimeout == "" && cfg.freqInterval != 0 {
		// periodic tasks require a timeout
		cfg.execTimeout = cfg.freqInterval
	}
	if cfg.ExecTimeout != "" {
		execTimeout, err := utils.GetTimeout(cfg.ExecTimeout)
		if err != nil {
			return fmt.Errorf("could not parse `timeout` for job %s: %v", cfg.Name, err)
		}
		if execTimeout < time.Duration(1*time.Millisecond) {
			// if there's no timeout set, that's ok, but if we have a timeout
			// set we need to make sure it's functional
			return fmt.Errorf("timeout '%v' cannot be less than 1ms", cfg.ExecTimeout)
		}
		cfg.execTimeout = execTimeout
	}
	if cfg.Exec != nil {
		cmd, err := commands.NewCommand(cfg.Exec, cfg.execTimeout,
			log.Fields{"job": cfg.Name})
		if err != nil {
			return fmt.Errorf("could not parse `exec` for job %s: %s", cfg.Name, err)
		}
		if cfg.Name == "" {
			cfg.Name = cmd.Exec
		}
		cmd.Name = cfg.Name
		cfg.exec = cmd
	}
	return nil
}

func (cfg *Config) validateRestarts() error {

	// defaults if omitted
	if cfg.Restarts == nil {
		cfg.restartLimit = 0
		return nil
	}

	const msg = `invalid 'restarts' field "%v": accepts positive integers, "unlimited", or "never"`

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

	return nil
}

// AddDiscoveryConfig validates the configuration for service discovery
// and attaches the discovery.Backend to it
func (cfg *Config) AddDiscoveryConfig(disc discovery.Backend) error {
	interfaces, ifaceErr := utils.ToStringArray(cfg.Interfaces)
	if ifaceErr != nil {
		return ifaceErr
	}
	ipAddress, err := utils.GetIP(interfaces)
	if err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	id := fmt.Sprintf("%s-%s", cfg.Name, hostname)

	cfg.discoveryCatalog = disc

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
		ID:           id,
		Name:         cfg.Name,
		Port:         cfg.Port,
		TTL:          cfg.TTL,
		Tags:         cfg.Tags,
		IPAddress:    ipAddress,
		ConsulExtras: consulExtras,
	}
	return nil
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (cfg *Config) String() string {
	return "jobs.Config[" + cfg.Name + "]"
}
