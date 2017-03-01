package watches

import (
	"context"
	"fmt"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
)

type Watch struct {
	Args    []string
	Command *commands.Command
	ID      string
	Poll    int

	// Event handling
	events.EventHandler
	startupEvent   events.Event
	startupTimeout int
	heartbeat      int
}

func (watch *Watch) Run() {
	// TODO: this will probably be a background context b/c we've got
	// message-passing to the main loop for cancellation
	ctx, cancel := context.WithCancel(context.TODO())

	timerSource := fmt.Sprintf("%s-watch-timer", watch.ID)
	events.NewEventTimer(ctx, watch.Rx,
		time.Duration(watch.heartbeat)*time.Second, timerSource)

	go func() {
		select {
		case event := <-watch.Rx:
			switch event.Code {
			case events.TimerExpired:
				if event.Source == timerSource {
					fmt.Printf("checking: %s\n", watch.ID)
					watch.Bus.Publish(
						events.Event{Code: events.StatusChanged, Source: watch.ID})
				}
			case events.Quit:
				if event.Source != watch.ID {
					break
				}
				fallthrough
			case events.Shutdown:
				watch.Unsubscribe(watch.Bus)
				close(watch.Rx)
				cancel()
				watch.Flush <- true
				return
			case watch.startupEvent.Code:
				// run this in a goroutine and pass it our context
				watch.Bus.Publish(
					events.Event{Code: events.Started, Source: watch.ID})
				fmt.Println("watch exec running!")
				watch.Bus.Publish(
					events.Event{Code: events.ExitSuccess, Source: watch.ID})
			default:
				fmt.Println("don't care about this message")
			}
		}
	}()
}
