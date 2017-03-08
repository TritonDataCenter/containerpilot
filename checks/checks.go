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
	Name           string
	exec           *commands.Command
	startupEvent   events.Event
	startupTimeout time.Duration
	poll           time.Duration

	events.EventHandler // Event handling
}

// NewHealthCheck ...
func NewHealthCheck(cfg *HealthCheckConfig) (*HealthCheck, error) {
	evt := events.Event{Code: events.StatusChanged, Source: cfg.Name}
	check := &HealthCheck{
		Name:           cfg.Name,
		exec:           cfg.exec,
		poll:           cfg.pollInterval,
		startupEvent:   evt,
		startupTimeout: -1,
	}
	check.Rx = make(chan events.Event, eventBufferSize)
	check.Flush = make(chan bool)
	return check, nil
}

// CheckHealth runs the health check and returns any error
func (check *HealthCheck) CheckHealth(ctx context.Context) error {
	// TODO: we want to update Run... functions to accept
	// a parent context so we can cancel them from this main loop
	return commands.RunWithTimeout(check.exec, log.Fields{
		"process": check.startupEvent.Code, "check": check.Name})
}

func (check *HealthCheck) Run(bus *events.EventBus) {
	check.Subscribe(bus)
	check.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	timerSource := fmt.Sprintf("%s-check-timer", check.Name)
	events.NewEventTimer(ctx, check.Rx, check.poll, timerSource)

	go func() {
		for {
			event := <-check.Rx
			switch event {
			case events.Event{events.TimerExpired, timerSource}:
				check.Bus.Publish(check.startupEvent)
			case
				events.Event{events.Quit, check.Name},
				events.Event{events.Quit, events.Closed},
				events.Event{events.Shutdown, events.Global}:
				check.Unsubscribe(check.Bus)
				close(check.Rx)
				cancel()
				check.Flush <- true
				return
			case check.startupEvent:
				err := check.CheckHealth(ctx)
				if err != nil {
					check.Bus.Publish(
						events.Event{Code: events.ExitSuccess, Source: check.Name})
				} else {
					check.Bus.Publish(
						events.Event{Code: events.ExitSuccess, Source: check.Name})
				}
			}
		}
	}()
}
