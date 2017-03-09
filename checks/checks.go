package checks

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
)

const eventBufferSize = 1000

type HealthCheck struct {
	Name string
	exec *commands.Command
	poll time.Duration

	events.EventHandler // Event handling
}

// NewHealthCheck ...
func NewHealthCheck(cfg *HealthCheckConfig) (*HealthCheck, error) {
	check := &HealthCheck{
		Name: cfg.Name,
		exec: cfg.exec,
		poll: cfg.pollInterval,
	}
	check.Rx = make(chan events.Event, eventBufferSize)
	check.Flush = make(chan bool)
	return check, nil
}

// CheckHealth runs the health check
func (check *HealthCheck) CheckHealth(ctx context.Context) {
	// TODO: what log fields do we really want here?
	check.exec.Run(ctx, check.Bus, log.Fields{
		"process": check.exec.Name, "check": check.Name})
}

// Run executes the event loop for the HealthCheck
func (check *HealthCheck) Run(bus *events.EventBus) {
	check.Subscribe(bus)
	check.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	pollSource := fmt.Sprintf("%s-poll", check.Name)
	events.NewEventTimer(ctx, check.Rx, check.poll, pollSource)

	go func() {
		for {
			event := <-check.Rx
			switch event {
			case events.Event{events.TimerExpired, pollSource}:
				check.CheckHealth(ctx)
			case
				events.Event{events.Quit, check.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				check.Unsubscribe(check.Bus)
				close(check.Rx)
				cancel()
				check.Flush <- true
				return
			}
		}
	}()
}
