package services

/*
TODO: this entire file will be eliminated after we update config syntax
*/

import (
	"fmt"
	"time"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/utils"
)

// TaskConfig configures tasks that run periodically
type TaskConfig struct {
	Name         string      `mapstructure:"name"`
	Exec         interface{} `mapstructure:"command"`
	Frequency    string      `mapstructure:"frequency"`
	Timeout      string      `mapstructure:"timeout"`
	timeout      time.Duration
	freqDuration time.Duration
}

// NewTaskConfigs parses json config into a validated slice of ServiceConfigs
func NewTaskConfigs(raw []interface{}) ([]*ServiceConfig, error) {
	var (
		tasks    []*TaskConfig
		services []*ServiceConfig
	)
	if raw == nil {
		return services, nil
	}
	if err := utils.DecodeRaw(raw, &tasks); err != nil {
		return nil, fmt.Errorf("task configuration error: %v", err)
	}
	for _, cfg := range tasks {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		service := cfg.ToServiceConfig()
		if err := service.Validate(nil); err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}

// ToServiceConfig ...
func (task *TaskConfig) ToServiceConfig() *ServiceConfig {
	service := &ServiceConfig{
		Exec:         task.Exec,
		Name:         task.Name,
		execTimeout:  task.timeout,
		restartLimit: unlimitedRestarts,
		freqInterval: task.freqDuration,
		startupEvent: events.Event{events.TimerExpired, task.Name}, // TODO: probably not the event we want here
	}
	return service
}

// Validate ...
func (task *TaskConfig) Validate() error {
	if task.Exec == nil {
		return fmt.Errorf("task did not provide a command")
	}

	// parse task frequency and ensure we have a valid value
	freq, err := utils.ParseDuration(task.Frequency)
	if err != nil {
		return fmt.Errorf("unable to parse frequency %s: %v", task.Frequency, err)
	}
	if freq < time.Millisecond {
		return fmt.Errorf("frequency %v cannot be less that %v", freq, taskMinDuration)
	}
	task.freqDuration = freq

	// parse task timeout and ensure we have a valid value
	if task.Timeout == "" {
		task.Timeout = task.Frequency
	}
	timeout, err := utils.ParseDuration(task.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse timeout %s: %v", task.Timeout, err)
	}
	if timeout < time.Duration(taskMinDuration) {
		return fmt.Errorf("timeout %v cannot be less that %v", task.Timeout, taskMinDuration)
	}
	task.timeout = timeout
	return nil
}
