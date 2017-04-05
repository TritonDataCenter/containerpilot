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
	Heartbeat         int           `mapstructure:"poll"` // time in seconds
	TTL               int           `mapstructure:"ttl"`
	Port              int           `mapstructure:"port"`
	Interfaces        interface{}   `mapstructure:"interfaces"`
	Tags              []string      `mapstructure:"tags"`
	ConsulConfig      *ConsulConfig `mapstructure:"consul"`
	heartbeatInterval time.Duration
	discoveryCatalog  discovery.Backend
	definition        *discovery.ServiceDefinition

	// timeouts and restarts
	ExecTimeout  string      `mapstructure:"execTimeout"`
	Restarts     interface{} `mapstructure:"restarts"`
	Frequency    string      `mapstructure:"frequency"`
	execTimeout  time.Duration
	exec         *commands.Command
	restartLimit int
	freqInterval time.Duration

	// related jobs
	PreStartExec    interface{} `mapstructure:"preStart"`
	PreStopExec     interface{} `mapstructure:"preStop"`
	PostStopExec    interface{} `mapstructure:"postStop"`
	startupEvent    events.Event
	startupTimeout  time.Duration
	stoppingEvent   events.Event
	stoppingTimeout time.Duration

	/* TODO v3:
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

// NewConfigs parses json config into a validated slice of Configs
func NewConfigs(raw []interface{}, disc discovery.Backend) ([]*Config, error) {
	var jobs []*Config
	if raw == nil {
		return jobs, nil
	}
	if err := utils.DecodeRaw(raw, &jobs); err != nil {
		return nil, fmt.Errorf("service configuration error: %v", err)
	}
	for _, job := range jobs {
		if err := job.Validate(disc); err != nil {
			return nil, err
		}
	}
	for _, job := range jobs {
		if job.PreStartExec != nil {
			preStart, err := NewPreStartConfig(job.Name, job.PreStartExec)
			if err != nil {
				return nil, err
			}
			job.setStartup(events.Event{events.ExitSuccess, preStart.Name}, 0)
			jobs = append(jobs, preStart)
		}
		if job.PreStopExec != nil {
			preStop, err := NewPreStopConfig(job.Name, job.PreStopExec)
			if err != nil {
				return nil, err
			}
			preStop.setStartup(events.Event{events.Stopping, job.Name}, 0)
			job.setStopping(events.Event{events.Stopped, preStop.Name}, 0)
			jobs = append(jobs, preStop)
		}
		if job.PostStopExec != nil {
			postStop, err := NewPostStopConfig(job.Name, job.PostStopExec)
			if err != nil {
				return nil, err
			}
			postStop.setStartup(events.Event{events.Stopped, job.Name}, 0)
			jobs = append(jobs, postStop)
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
	if err := cfg.validateRestarts(); err != nil {
		return err
	}
	if err := cfg.validateExec(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) setStartup(evt events.Event, timeout time.Duration) {
	cfg.startupEvent = evt
	cfg.startupTimeout = timeout
}

func (cfg *Config) setStopping(evt events.Event, timeout time.Duration) {
	cfg.stoppingEvent = evt
	cfg.stoppingTimeout = timeout
}

func (cfg *Config) validateDiscovery(disc discovery.Backend) error {
	// if port isn't set then we won't do any discovery for this job
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
	// TODO v3: these will be exposed as config values when we do the
	// config update. for now we set defaults here
	cfg.startupTimeout = 0
	cfg.startupEvent = events.GlobalStartup
	cfg.stoppingTimeout = 0
	cfg.stoppingEvent = events.NonEvent
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
