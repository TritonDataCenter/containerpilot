package events

import (
	"context"
	"fmt"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
)

type HealthCheck struct {
	Args    []string
	Command *commands.Command
	ID      string
	Poll    int

	// Event handling
	events.EventHandler
	startupEvent   events.Event
	startupTimeout int
	restarts       int
	heartbeat      int
}

func (check *HealthCheck) Run() {
	// TODO: this will probably be a background context b/c we've got
	// message-passing to the main loop for cancellation
	ctx, cancel := context.WithCancel(context.TODO())

	timerSource := fmt.Sprintf("%s-check-timer", check.ID)
	events.NewEventTimer(ctx, check.Rx,
		time.Duration(check.heartbeat)*time.Second, timerSource)

	go func() {
		select {
		case event := <-check.Rx:
			switch event.Code {
			case events.TimerExpired:
				if event.Source == timerSource {
					fmt.Printf("checking: %s\n", check.ID)
					check.Bus.Publish(
						events.Event{Code: events.StatusChanged, Source: check.ID})
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
				// run this in a goroutine and pass it our context
				check.Bus.Publish(
					events.Event{Code: events.Started, Source: check.ID})
				fmt.Println("check exec running!")
				check.Bus.Publish(
					events.Event{Code: events.ExitSuccess, Source: check.ID})
			default:
				fmt.Println("don't care about this message")
			}
		}
	}()
}
