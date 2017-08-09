package watches

import (
	"context"
	"fmt"
	"time"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
)

// Watch represents an event to signal when something changes
type Watch struct {
	Name             string
	serviceName      string
	tag              string
	dc               string
	poll             int
	discoveryService discovery.Backend

	events.EventHandler // Event handling
}

// NewWatch creates a Watch from a validated Config
func NewWatch(cfg *Config) *Watch {
	watch := &Watch{
		Name:             cfg.Name,
		serviceName:      cfg.serviceName,
		tag:              cfg.Tag,
		dc:               cfg.DC,
		poll:             cfg.Poll,
		discoveryService: cfg.discoveryService,
	}
	watch.InitRx()
	return watch
}

// FromConfigs creates Watches from a slice of validated Configs
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
func (watch *Watch) CheckForUpstreamChanges() (bool, bool) {
	return watch.discoveryService.CheckForUpstreamChanges(watch.serviceName, watch.tag, watch.dc)
}

// Run executes the event loop for the Watch
func (watch *Watch) Run(bus *events.EventBus) {
	watch.Subscribe(bus)
	watch.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	timerSource := fmt.Sprintf("%s.poll", watch.Name)
	events.NewEventTimer(ctx, watch.Rx,
		time.Duration(watch.poll)*time.Second, timerSource)

	go func() {
		defer func() {
			cancel()
			watch.Unsubscribe(watch.Bus)
		}()
		for {
			select {
			case event, ok := <-watch.Rx:
				if !ok {
					return
				}
				switch event {
				case events.Event{events.TimerExpired, timerSource}:
					didChange, isHealthy := watch.CheckForUpstreamChanges()
					if didChange {
						watch.Bus.Publish(events.Event{events.StatusChanged, watch.Name})
						// we only send the StatusHealthy and StatusUnhealthy
						// events if there was a change
						if isHealthy {
							watch.Bus.Publish(events.Event{events.StatusHealthy, watch.Name})
						} else {
							watch.Bus.Publish(events.Event{events.StatusUnhealthy, watch.Name})
						}
					}
				case
					events.Event{events.Quit, watch.Name},
					events.QuitByClose,
					events.GlobalShutdown:
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (watch *Watch) String() string {
	return "watches.Watch[" + watch.Name + "]"
}
