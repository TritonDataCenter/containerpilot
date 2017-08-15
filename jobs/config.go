package jobs

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/config/decode"
	"github.com/joyent/containerpilot/config/services"
	"github.com/joyent/containerpilot/config/timing"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
	log "github.com/sirupsen/logrus"
)

const taskMinDuration = time.Millisecond

// Config holds the configuration for service discovery data
type Config struct {
	Name string      `mapstructure:"name"`
	Exec interface{} `mapstructure:"exec"`

	// service discovery
	Port              int           `mapstructure:"port"`
	Interfaces        interface{}   `mapstructure:"interfaces"`
	Tags              []string      `mapstructure:"tags"`
	ConsulExtras      *ConsulExtras `mapstructure:"consul"`
	serviceDefinition *discovery.ServiceDefinition

	// health checking
	Health            *HealthConfig `mapstructure:"health"`
	healthCheckExec   *commands.Command
	heartbeatInterval time.Duration
	ttl               int

	// timeouts and restarts
	ExecTimeout     string      `mapstructure:"timeout"`
	Restarts        interface{} `mapstructure:"restarts"`
	StopTimeout     string      `mapstructure:"stopTimeout"`
	execTimeout     time.Duration
	exec            *commands.Command
	stoppingTimeout time.Duration
	restartLimit    int
	freqInterval    time.Duration

	// related jobs and frequency
	When              *WhenConfig `mapstructure:"when"`
	whenEvent         events.Event
	whenTimeout       time.Duration
	whenStartsLimit   int
	stoppingWaitEvent events.Event
}

// WhenConfig determines when a Job runs (dependencies on other Jobs,
// Watches, or frequency timers)
type WhenConfig struct {
	Frequency string `mapstructure:"interval"`
	Source    string `mapstructure:"source"`
	Once      string `mapstructure:"once"`
	Each      string `mapstructure:"each"`
	Timeout   string `mapstructure:"timeout"`
}

// HealthConfig configures the Job's health checks
type HealthConfig struct {
	CheckExec    interface{} `mapstructure:"exec"`
	CheckTimeout string      `mapstructure:"timeout"`
	Heartbeat    int         `mapstructure:"interval"` // time in seconds
	TTL          int         `mapstructure:"ttl"`      // time in seconds
}

// ConsulExtras handles additional Consul configuration.
type ConsulExtras struct {
	EnableTagOverride              bool   `mapstructure:"enableTagOverride"`
	DeregisterCriticalServiceAfter string `mapstructure:"deregisterCriticalServiceAfter"`
}

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}, disc discovery.Backend) ([]*Config, error) {
	var jobs []*Config
	if raw == nil {
		return jobs, nil
	}
	if err := decode.ToStruct(raw, &jobs); err != nil {
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
	if err := cfg.validateDiscovery(disc); err != nil {
		return err
	}
	if err := cfg.validateWhen(); err != nil {
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
	// setting up discovery requires the TTL from the health check first
	if err := cfg.validateHealthCheck(); err != nil {
		return err
	}
	// if port isn't set then we won't do any discovery for this job
	if (cfg.Port == 0 || disc == nil) && cfg.Name != "" {
		return nil
	}
	// we only need to validate the name if we're doing discovery;
	// we'll just take the name of the exec otherwise
	if err := services.ValidateName(cfg.Name); err != nil {
		return err
	}
	return cfg.addDiscoveryConfig(disc)
}

func (cfg *Config) validateWhen() error {
	if cfg.When == nil {
		// set defaults (frequencyInterval will be zero-value)
		cfg.When = &WhenConfig{} // give us a safe zero-value
		cfg.whenTimeout = time.Duration(0)
		cfg.whenEvent = events.GlobalStartup
		cfg.whenStartsLimit = 1
		return nil
	}

	if (cfg.When.Frequency != "" && cfg.When.Once != "") ||
		(cfg.When.Frequency != "" && cfg.When.Each != "") ||
		(cfg.When.Once != "" && cfg.When.Each != "") {
		return fmt.Errorf("job[%s].when can have only one of 'interval', 'once', or 'each'",
			cfg.Name)
	}
	if cfg.When.Frequency != "" {
		return cfg.validateFrequency()
	}
	return cfg.validateWhenEvent()
}

func (cfg *Config) validateFrequency() error {
	freq, err := timing.ParseDuration(cfg.When.Frequency)
	if err != nil {
		return fmt.Errorf("unable to parse job[%s].when.interval '%s': %v",
			cfg.Name, cfg.When.Frequency, err)
	}
	if freq < taskMinDuration {
		return fmt.Errorf("job[%s].when.interval '%s' cannot be less than %v",
			cfg.Name, cfg.When.Frequency, taskMinDuration)
	}
	cfg.freqInterval = freq
	cfg.whenTimeout = time.Duration(0)
	cfg.whenEvent = events.GlobalStartup
	cfg.whenStartsLimit = 1
	return nil
}

func (cfg *Config) validateWhenEvent() error {

	whenTimeout, err := timing.GetTimeout(cfg.When.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse job[%s].when.timeout: %v",
			cfg.Name, err)
	}
	cfg.whenTimeout = whenTimeout

	var eventCode events.EventCode
	if cfg.When.Once != "" {
		eventCode, err = events.FromString(cfg.When.Once)
		cfg.whenStartsLimit = 1
	} else {
		eventCode, err = events.FromString(cfg.When.Each)
		cfg.whenStartsLimit = unlimited
	}
	if err != nil {
		return fmt.Errorf("unable to parse job[%s].when.event: %v",
			cfg.Name, err)
	}
	cfg.whenEvent = events.Event{eventCode, cfg.When.Source}
	return nil
}

func (cfg *Config) validateStoppingTimeout() error {
	stoppingTimeout, err := timing.GetTimeout(cfg.StopTimeout)
	if err != nil {
		return fmt.Errorf("unable to parse job[%s].stopTimeout '%s': %v",
			cfg.Name, cfg.StopTimeout, err)
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
		execTimeout, err := timing.GetTimeout(cfg.ExecTimeout)
		if err != nil {
			return fmt.Errorf("unable to parse job[%s].timeout '%s': %v",
				cfg.Name, cfg.ExecTimeout, err)
		}
		if execTimeout < time.Duration(1*time.Millisecond) {
			// if there's no timeout set, that's ok, but if we have a timeout
			// set we need to make sure it's functional
			return fmt.Errorf("job[%s].timeout '%v' cannot be less than 1ms",
				cfg.Name, cfg.ExecTimeout)
		}
		cfg.execTimeout = execTimeout
	}
	if cfg.Exec != nil {
		cmd, err := commands.NewCommand(cfg.Exec, cfg.execTimeout,
			log.Fields{"job": cfg.Name})
		if err != nil {
			return fmt.Errorf("unable to create job[%s].exec: %v", cfg.Name, err)
		}
		if cfg.Name == "" {
			cfg.Name = cmd.Exec
		}
		cmd.Name = cfg.Name
		cfg.exec = cmd
	}
	return nil
}

func (cfg *Config) validateHealthCheck() error {
	if cfg.Port != 0 && cfg.Health == nil && cfg.Name != "containerpilot" {
		return fmt.Errorf("job[%s].health must be set if 'port' is set", cfg.Name)
	}
	if cfg.Health == nil {
		return nil // non-advertised jobs don't need health checks
	}
	if cfg.Health.Heartbeat < 1 {
		return fmt.Errorf("job[%s].health.interval must be > 0", cfg.Name)
	}
	if cfg.Health.TTL < 1 {
		return fmt.Errorf("job[%s].health.ttl must be > 0", cfg.Name)
	}

	cfg.ttl = cfg.Health.TTL
	cfg.heartbeatInterval = time.Duration(cfg.Health.Heartbeat) * time.Second

	var checkTimeout time.Duration
	if cfg.Health.CheckTimeout != "" {
		parsedTimeout, err := timing.GetTimeout(cfg.Health.CheckTimeout)
		if err != nil {
			return fmt.Errorf("could not parse job[%s].health.timeout '%s': %v",
				cfg.Name, cfg.Health.CheckTimeout, err)
		}
		checkTimeout = parsedTimeout
	} else {
		checkTimeout = cfg.execTimeout
	}

	if cfg.Health.CheckExec != nil {
		// the telemetry service won't have a health check
		checkName := "check." + cfg.Name
		cmd, err := commands.NewCommand(cfg.Health.CheckExec, checkTimeout,
			log.Fields{"check": checkName})
		if err != nil {
			return fmt.Errorf("unable to create job[%s].health.exec: %v",
				cfg.Name, err)
		}
		cmd.Name = checkName
		cfg.healthCheckExec = cmd
	}
	return nil
}

func (cfg *Config) validateRestarts() error {

	// defaults if omitted
	if cfg.Restarts == nil {
		if cfg.freqInterval != time.Duration(0) {
			cfg.restartLimit = unlimited
		} else {
			cfg.restartLimit = 0
		}
		return nil
	}
	const msg = `job[%s].restarts field '%v' invalid: %v`

	switch t := cfg.Restarts.(type) {
	case string:
		if t == "unlimited" {
			if cfg.When.Each != "" {
				return fmt.Errorf(msg, cfg.Name, cfg.Restarts,
					`may not be used when 'job.when.each' is set because it may result in infinite processes`)
			}
			cfg.restartLimit = unlimited
		} else if t == "never" {
			cfg.restartLimit = 0
		} else if i, err := strconv.Atoi(t); err == nil && i >= 0 {
			cfg.restartLimit = i
		} else {
			return fmt.Errorf(msg, cfg.Name, cfg.Restarts,
				`accepts positive integers, "unlimited", or "never"`)
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
			return fmt.Errorf(msg, cfg.Name, cfg.Restarts,
				`number must be positive integer`)
		}
	default:
		return fmt.Errorf(msg, cfg.Name, cfg.Restarts,
			`accepts positive integers, "unlimited", or "never"`)
	}

	return nil
}

// addDiscoveryConfig validates the configuration for service discovery
// and attaches the discovery.ServiceDefinition to the Config
func (cfg *Config) addDiscoveryConfig(disc discovery.Backend) error {
	interfaces, ifaceErr := decode.ToStrings(cfg.Interfaces)
	if ifaceErr != nil {
		return ifaceErr
	}
	ipAddress, err := services.GetIP(interfaces)
	if err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	id := fmt.Sprintf("%s-%s", cfg.Name, hostname)

	var (
		enableTagOverride bool
		deregAfter        string
	)

	if cfg.ConsulExtras != nil {
		deregAfter = cfg.ConsulExtras.DeregisterCriticalServiceAfter
		_, err := time.ParseDuration(deregAfter)
		if err != nil {
			return fmt.Errorf(
				"unable to parse job[%s].consul.deregisterCriticalServiceAfter: %s",
				cfg.Name, err)
		}
		enableTagOverride = cfg.ConsulExtras.EnableTagOverride
	}
	cfg.serviceDefinition = &discovery.ServiceDefinition{
		ID:                             id,
		Name:                           cfg.Name,
		Port:                           cfg.Port,
		TTL:                            cfg.ttl,
		Tags:                           cfg.Tags,
		IPAddress:                      ipAddress,
		DeregisterCriticalServiceAfter: deregAfter,
		EnableTagOverride:              enableTagOverride,
		Consul:                         disc,
	}
	return nil
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (cfg *Config) String() string {
	return "jobs.Config[" + cfg.Name + "]"
}
