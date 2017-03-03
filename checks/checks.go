package checks

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
)

type HealthCheck struct {
	ID   string
	exec *commands.Command

	// Event handling
	events.EventHandler
	startupEvent   events.Event
	startupTimeout int
	restarts       int
	poll           int
}

// NewHealthCheck ...
func NewHealthCheck(cfg *HealthCheckConfig) (*HealthCheck, error) {
	check := &HealthCheck{}
	check.ID = cfg.Name
	check.poll = cfg.Poll

	check.Rx = make(chan events.Event, 1000)
	check.Flush = make(chan bool)
	check.startupEvent = events.Event{Code: events.StatusChanged, Source: check.ID}
	check.startupTimeout = -1

	cmd, err := commands.NewCommand(cfg.HealthCheckExec, cfg.Timeout)
	if err != nil {
		// TODO: this is config syntax specific and should be updated
		return nil, fmt.Errorf("could not parse `health` in check %s: %s",
			cfg.Name, err)
	}
	check.exec = cmd
	return check, nil
}

// CheckHealth runs the health check and returns any error
func (check *HealthCheck) CheckHealth(ctx context.Context) error {
	// TODO: we want to update Run... functions to accept
	// a parent context so we can cancel them from this main loop
	return commands.RunWithTimeout(check.exec, log.Fields{
		"process": check.startupEvent.Code, "check": check.ID})
}

func (check *HealthCheck) Run() {
	// TODO: this will probably be a background context b/c we've got
	// message-passing to the main loop for cancellation
	ctx, cancel := context.WithCancel(context.TODO())

	timerSource := fmt.Sprintf("%s-check-timer", check.ID)
	events.NewEventTimer(ctx, check.Rx,
		time.Duration(check.poll)*time.Second, timerSource)

	go func() {
		select {
		case event := <-check.Rx:
			switch event.Code {
			case events.TimerExpired:
				if event.Source == timerSource {
					check.Bus.Publish(
						events.Event{Code: check.startupEvent.Code, Source: check.ID})
				}
			case events.Quit:
				if event.Source == check.ID {
					break
				}
				fallthrough
			case events.Shutdown:
				check.Unsubscribe(check.Bus)
				close(check.Rx)
				cancel()
				check.Flush <- true
				return
			case check.startupEvent.Code:
				if event.Source != check.ID {
					break
				}
				check.Bus.Publish(
					events.Event{Code: events.Started, Source: check.ID})
				err := check.CheckHealth(ctx)
				if err != nil {
					check.Bus.Publish(
						events.Event{Code: events.ExitSuccess, Source: check.ID})
				} else {
					check.Bus.Publish(
						events.Event{Code: events.ExitSuccess, Source: check.ID})
				}
			default:
				fmt.Println("don't care about this message")
			}
		}
	}()
}
