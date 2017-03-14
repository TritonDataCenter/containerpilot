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
	startupEvent     events.Event // TODO: probably want this to be a Service name?
	discoveryService discovery.Backend

	events.EventHandler // Event handling
}

// NewWatch ...
func NewWatch(cfg *Config) *Watch {
	watch := &Watch{
		Name:             cfg.Name,
		poll:             cfg.Poll,
		Tag:              cfg.Tag,
		exec:             cfg.exec,
		discoveryService: cfg.discoveryService,
	}
	watch.Rx = make(chan events.Event, eventBufferSize)
	watch.Flush = make(chan bool)
	return watch
}

// FromConfigs ...
func FromConfigs(cfgs []*Config) []*Watch {
	watches := []*Watch{}
	for _, cfg := range cfgs {
		watch := NewWatch(cfg)
		watches = append(watches, watch)
	}
	return watches
}

// CheckForUpstreamChanges checks the service discovery endpoint for any changes
// in a dependent backend. Returns true when there has been a change.
func (watch *Watch) CheckForUpstreamChanges() bool {
	return watch.discoveryService.CheckForUpstreamChanges(watch.Name, watch.Tag)
}

// OnChange runs the Watch's executable
func (watch *Watch) OnChange(ctx context.Context) {
	watch.exec.Run(ctx, watch.Bus, log.Fields{
		"process": watch.startupEvent.Code, "watch": watch.Name})
}

// Run executes the event loop for the Watch
func (watch *Watch) Run(bus *events.EventBus) {
	watch.Subscribe(bus)
	watch.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	timerSource := fmt.Sprintf("%s-watch-poll", watch.Name)
	events.NewEventTimer(ctx, watch.Rx,
		time.Duration(watch.poll)*time.Second, timerSource)
	log.Debug(timerSource)

	go func() {
		for {
			event := <-watch.Rx
			switch event {
			case events.Event{events.TimerExpired, timerSource}:
				changed := watch.CheckForUpstreamChanges()
				if changed {
					watch.OnChange(ctx)
				}
			case
				events.Event{events.Quit, watch.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				watch.Unsubscribe(watch.Bus)
				close(watch.Rx)
				cancel()
				watch.Flush <- true
				return
			}
		}
	}()
}
