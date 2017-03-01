package services

import (
	"context"
	"fmt"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
)

/*
TODO: this is temporary while I hack out how this will interact
with everything else. It'll go in the `services` package when I'm done
*/

type ServiceEvents struct {
	Args      []string
	Command   *commands.Command
	ID        string
	Heartbeat int
	Restart   string // TODO config
	Wait      int    // TODO config
	WaitOn    string // TODO config
	Status    bool   // TODO config

	// Event handling
	events.EventHandler
	startupEvent   events.Event
	startupTimeout int
	restarts       int // -1 for "always"
	runEvery       int
	heartbeat      int
}

func (svc *ServiceEvents) Run() {
	// TODO: this will probably be a background context b/c we've got
	// message-passing to the main loop for cancellation
	ctx, cancel := context.WithCancel(context.TODO())

	runEverySource := fmt.Sprintf("%s-run-every", svc.ID)
	if svc.runEvery > 0 {
		events.NewEventTimeout(ctx, svc.Rx,
			time.Duration(svc.runEvery)*time.Second, runEverySource)
	}

	heartbeatSource := fmt.Sprintf("%s-heartbeat", svc.ID)
	if svc.heartbeat > 0 {
		events.NewEventTimeout(ctx, svc.Rx,
			time.Duration(svc.heartbeat)*time.Second, heartbeatSource)
	}

	timeoutSource := fmt.Sprintf("%s-wait-timeout", svc.ID)
	if svc.startupTimeout > 0 {
		events.NewEventTimeout(ctx, svc.Rx,
			time.Duration(svc.startupTimeout)*time.Second, timeoutSource)
	}

	go func() {
		select {
		case event := <-svc.Rx:
			switch event.Code {
			case events.TimerExpired:
				switch event.Source {
				case heartbeatSource:
					if svc.Status == true {
						fmt.Printf("heartbeat: %s\n", svc.ID)
					}
				case timeoutSource:
					svc.Bus.Publish(events.Event{Code: events.TimerExpired, Source: svc.ID})
					svc.Rx <- events.Event{Code: events.Quit, Source: svc.ID}
				case runEverySource:
					if svc.restarts > 0 || svc.restarts < 0 {
						svc.Rx <- events.Event{Code: svc.startupEvent.Code, Source: svc.ID}
						svc.restarts--
					}
				}
			case events.Quit:
				if event.Source == svc.ID {
					break
				}
				fallthrough
			case events.Shutdown:
				svc.Unsubscribe(svc.Bus)
				close(svc.Rx)
				cancel()
				svc.Flush <- true
				return
			case events.ExitSuccess:
			case events.ExitFailed:
				if event.Source == svc.ID && svc.restarts > 0 || svc.restarts < 0 {
					svc.Rx <- events.Event{Code: svc.startupEvent.Code, Source: svc.ID}
					svc.restarts--
				}
			case svc.startupEvent.Code:
				// run this in a goroutine and pass it our context
				svc.Bus.Publish(events.Event{Code: events.Started, Source: svc.ID})
				fmt.Println("running!")
				svc.Bus.Publish(events.Event{Code: events.ExitSuccess, Source: svc.ID})
			default:
				fmt.Println("don't care about this message")
			}
		}
	}()
}
