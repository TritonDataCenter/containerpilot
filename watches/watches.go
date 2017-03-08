package watches

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
)

const eventBufferSize = 1000

// Watch represents a task to execute when something changes
type Watch struct {
	Name             string
	Tag              string
	exec             *commands.Command
	startupTimeout   int
	poll             int
	startupEvent     events.Event
	discoveryService discovery.Backend

	events.EventHandler // Event handling
}

func NewWatch(cfg *WatchConfig) (*Watch, error) {
	evt := events.Event{Code: events.StatusChanged, Source: cfg.Name}
	watch := &Watch{
		Name:             cfg.Name,
		poll:             cfg.Poll,
		Tag:              cfg.Tag,
		exec:             cfg.onChangeExec,
		discoveryService: cfg.discoveryService,
		startupEvent:     evt,
		startupTimeout:   -1,
	}
	watch.Rx = make(chan events.Event, eventBufferSize)
	watch.Flush = make(chan bool)
	return watch, nil
}

// CheckForUpstreamChanges checks the service discovery endpoint for any changes
// in a dependent backend. Returns true when there has been a change.
func (watch *Watch) CheckForUpstreamChanges() bool {
	return watch.discoveryService.CheckForUpstreamChanges(watch.Name, watch.Tag)
}

// OnChange runs the watch's executable, returning an error on failure.
func (watch *Watch) OnChange(ctx context.Context) error {
	// TODO: we want to update Run... functions to accept
	// a parent context so we can cancel them from this main loop
	return commands.RunWithTimeout(watch.exec, log.Fields{
		"process": watch.startupEvent.Code, "watch": watch.Name})
}

func (watch *Watch) Run(bus *events.EventBus) {
	watch.Subscribe(bus)
	watch.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	timerSource := fmt.Sprintf("%s-watch-timer", watch.Name)
	events.NewEventTimer(ctx, watch.Rx,
		time.Duration(watch.poll)*time.Second, timerSource)

	go func() {
		for {
			event := <-watch.Rx
			switch event {
			case events.Event{events.TimerExpired, timerSource}:
				changed := watch.CheckForUpstreamChanges()
				if changed {
					watch.Bus.Publish(watch.startupEvent)
				}
			case
				events.Event{events.Quit, watch.Name},
				events.Event{events.Quit, events.Closed},
				events.Event{events.Shutdown, events.Global}:
				watch.Unsubscribe(watch.Bus)
				close(watch.Rx)
				cancel()
				watch.Flush <- true
				return
			case watch.startupEvent:
				err := watch.OnChange(ctx)
				if err != nil {
					watch.Bus.Publish(
						events.Event{Code: events.ExitSuccess, Source: watch.Name})
				} else {
					watch.Bus.Publish(
						events.Event{Code: events.ExitSuccess, Source: watch.Name})
				}
			}
		}
	}()
}
