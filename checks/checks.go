package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
)

const eventBufferSize = 1000

// HealthCheck manages state of periodic health checks.
type HealthCheck struct {
	Name    string
	jobName string
	exec    *commands.Command
	poll    time.Duration

	events.EventHandler // Event handling
}

// NewHealthCheck creates a HealthCheck from a validated Config struct
func NewHealthCheck(cfg *Config) *HealthCheck {
	check := &HealthCheck{
		Name:    cfg.Name,
		jobName: cfg.jobName,
		exec:    cfg.exec,
		poll:    cfg.pollInterval,
	}
	check.Rx = make(chan events.Event, eventBufferSize)
	check.Flush = make(chan bool)
	return check
}

// FromConfigs creates HealthChecks from a slice of validated Configs
func FromConfigs(cfgs []*Config) []*HealthCheck {
	checks := []*HealthCheck{}
	for _, cfg := range cfgs {
		check := NewHealthCheck(cfg)
		checks = append(checks, check)
	}
	return checks
}

// CheckHealth runs the health check
func (check *HealthCheck) CheckHealth(ctx context.Context) {
	check.exec.Run(ctx, check.Bus)
}

// Run executes the event loop for the HealthCheck
func (check *HealthCheck) Run(bus *events.EventBus) {
	check.Subscribe(bus)
	check.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	pollSource := fmt.Sprintf("%s.poll", check.Name)
	events.NewEventTimer(ctx, check.Rx, check.poll, pollSource)

	go func() {
		for {
			event := <-check.Rx
			switch event {
			case events.Event{events.TimerExpired, pollSource}:
				check.CheckHealth(ctx)
			case events.Event{events.ExitSuccess, check.Name}:
				check.Bus.Publish(events.Event{events.StatusHealthy, check.jobName})
			case events.Event{events.ExitFailed, check.Name}:
				check.Bus.Publish(events.Event{events.StatusUnhealthy, check.jobName})
			case
				events.Event{events.Quit, check.Name},
				events.Event{events.Stopped, check.jobName},
				events.QuitByClose,
				events.GlobalShutdown:
				check.Unsubscribe(check.Bus)
				close(check.Rx)
				cancel()
				check.exec.CloseLogs()
				check.Flush <- true
				return
			}
		}
	}()
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (check *HealthCheck) String() string {
	return "HealthCheck[%v]" + check.Name
}
