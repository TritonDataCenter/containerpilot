// Package watches manages the configuration and running of Consul
// service monitoring
package watches

import (
	"context"
	"time"

	"github.com/tritondatacenter/containerpilot/discovery"
	"github.com/tritondatacenter/containerpilot/events"
)

// Watch represents an event to signal when something changes
type Watch struct {
	Name             string
	serviceName      string
	tag              string
	dc               string
	poll             int
	discoveryService discovery.Backend
	rx               chan events.Event

	events.Publisher
}

const eventBufferSize = 1000

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
	// watch.InitRx()
	watch.rx = make(chan events.Event, eventBufferSize)
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

// Tick returns the watcher's ticker time duration.
func (watch *Watch) Tick() time.Duration {
	return time.Duration(watch.poll) * time.Second
}

// Run executes the event loop for the Watch
func (watch *Watch) Run(pctx context.Context, bus *events.EventBus) {
	watch.Register(bus)
	ctx, cancel := context.WithCancel(pctx)
	timerSource := watch.Name + ".poll"

	// TODO(justinwr@): this could be replaced by a simple Ticker
	events.NewEventTimer(ctx, watch.rx, watch.Tick(), timerSource)

	go func() {
		defer func() {
			cancel()
			watch.Unregister()
			watch.Wait()
		}()
		for {
			select {
			case event, ok := <-watch.rx:
				if !ok || event == events.QuitByTest {
					return
				}
				if event == (events.Event{Code: events.TimerExpired, Source: timerSource}) {
					didChange, isHealthy := watch.CheckForUpstreamChanges()
					if didChange {
						watch.Publish(events.Event{Code: events.StatusChanged, Source: watch.Name})
						// we only send the StatusHealthy and StatusUnhealthy
						// events if there was a change
						if isHealthy {
							watch.Publish(events.Event{Code: events.StatusHealthy, Source: watch.Name})
						} else {
							watch.Publish(events.Event{Code: events.StatusUnhealthy, Source: watch.Name})
						}
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Receive receives an event into the internal control channel.
func (watch *Watch) Receive(event events.Event) {
	watch.rx <- event
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (watch *Watch) String() string {
	return "watches.Watch[" + watch.Name + "]"
}
