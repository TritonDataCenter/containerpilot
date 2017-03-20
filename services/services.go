package services

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
)

// Some magic numbers used internally by restart limits
const (
	unlimitedRestarts = -1
	eventBufferSize   = 1000
)

// Service configures the service discovery data
type Service struct {
	Name             string
	exec             *commands.Command
	Status           bool // TODO: we'll need this to carry more info than bool
	discoveryService discovery.Backend
	Definition       *discovery.ServiceDefinition

	// related events
	startupEvent    events.Event
	startupTimeout  time.Duration
	stoppingEvent   events.Event
	stoppingTimeout time.Duration

	// timing and restarts
	heartbeat      time.Duration
	restartLimit   int
	restartsRemain int
	frequency      time.Duration

	events.EventHandler // Event handling
}

// NewService creates a new Service from a Config
func NewService(cfg *Config) *Service {
	service := &Service{
		Name:             cfg.Name,
		exec:             cfg.exec,
		heartbeat:        cfg.heartbeatInterval,
		discoveryService: cfg.discoveryService,
		Definition:       cfg.definition,
		startupEvent:     cfg.startupEvent,
		startupTimeout:   cfg.startupTimeout,
		stoppingEvent:    cfg.stoppingEvent,
		stoppingTimeout:  cfg.stoppingTimeout,
		restartLimit:     cfg.restartLimit,
		restartsRemain:   cfg.restartLimit,
		frequency:        cfg.freqInterval,
	}
	service.Rx = make(chan events.Event, eventBufferSize)
	service.Flush = make(chan bool)
	return service
}

// FromConfigs ...
func FromConfigs(cfgs []*Config) []*Service {
	services := []*Service{}
	for _, cfg := range cfgs {
		service := NewService(cfg)
		services = append(services, service)
	}
	return services
}

// SendHeartbeat sends a heartbeat for this service
func (svc *Service) SendHeartbeat() {
	if svc.discoveryService != nil || svc.Definition != nil {
		svc.discoveryService.SendHeartbeat(svc.Definition)
	}
}

// MarkForMaintenance marks this service for maintenance
func (svc *Service) MarkForMaintenance() {
	if svc.discoveryService != nil || svc.Definition != nil {
		svc.discoveryService.MarkForMaintenance(svc.Definition)
	}
}

// Deregister will deregister this instance of the service
func (svc *Service) Deregister() {
	if svc.discoveryService != nil || svc.Definition != nil {
		svc.discoveryService.Deregister(svc.Definition)
	}
}

// StartService runs the Service's executable
func (svc *Service) StartService(ctx context.Context) {
	if svc.exec != nil {
		svc.exec.Run(ctx, svc.Bus, log.Fields{
			"process": svc.startupEvent.Code, "id": svc.Name})
	}
}

// Kill sends SIGTERM to the Service's executable, if any
func (svc *Service) Kill() {
	if svc.exec != nil {
		if svc.exec.Cmd != nil {
			if svc.exec.Cmd.Process != nil {
				svc.exec.Cmd.Process.Kill()
			}
		}
	}
}

// Run executes the event loop for the Service
func (svc *Service) Run(bus *events.EventBus) {
	if svc.exec == nil {
		// temporary: after config update having nil exec will be
		// an error
		return
	}

	svc.Subscribe(bus)
	svc.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	runEverySource := fmt.Sprintf("%s-run-every", svc.Name)
	if svc.frequency > 0 {
		events.NewEventTimer(ctx, svc.Rx, svc.frequency, runEverySource)
	}

	heartbeatSource := fmt.Sprintf("%s-heartbeat", svc.Name)
	if svc.heartbeat > 0 {
		events.NewEventTimer(ctx, svc.Rx, svc.heartbeat, heartbeatSource)
	}

	startTimeoutSource := fmt.Sprintf("%s-wait-timeout", svc.Name)
	if svc.startupTimeout > 0 {
		events.NewEventTimeout(ctx, svc.Rx, svc.startupTimeout, startTimeoutSource)
	}

	go func() {
	loop: // aw yeah, goto like it's 1968!
		for {
			event := <-svc.Rx
			switch event {
			case events.Event{events.TimerExpired, heartbeatSource}:
				// non-advertised services shouldn't receive this event
				// but we'll hit a null-pointer if we screw it up
				if svc.Status == true && svc.Definition != nil {
					svc.SendHeartbeat()
				}
			case events.Event{events.TimerExpired, startTimeoutSource}:
				svc.Bus.Publish(events.Event{
					Code: events.TimerExpired, Source: svc.Name})
				svc.Rx <- events.Event{Code: events.Quit, Source: svc.Name}
			case events.Event{events.TimerExpired, runEverySource}:
				if !svc.restartPermitted() {
					log.Debugf("restart not permitted: %v", svc.Name)
					break loop
				}
				svc.restartsRemain--
				svc.Rx <- svc.startupEvent
			case events.Event{events.StatusUnhealthy, svc.Name}:
				// TODO: add a "SendFailedHeartbeat" method
				svc.Status = false
			case events.Event{events.StatusHealthy, svc.Name}:
				svc.Status = true
			case
				events.Event{events.Quit, svc.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				break loop
			case
				events.Event{events.ExitSuccess, svc.Name},
				events.Event{events.ExitFailed, svc.Name}:
				if svc.frequency > 0 {
					break // note: breaks switch only
				}
				if !svc.restartPermitted() {
					log.Debugf("restart not permitted: %v", svc.Name)
					break loop
				}
				svc.restartsRemain--
				svc.Rx <- svc.startupEvent
			case svc.startupEvent:
				svc.StartService(ctx)
			}
		}
		svc.cleanup(ctx, cancel)
	}()
}

func (svc *Service) restartPermitted() bool {
	if svc.restartLimit == unlimitedRestarts || svc.restartsRemain > 0 {
		return true
	}
	return false
}

// cleanup fires the Stopping event and will wait to receive a stoppingEvent
// if one is configured. cleans up registration to event bus and closes all
// channels and contexts when done.
func (svc *Service) cleanup(ctx context.Context, cancel context.CancelFunc) {
	stoppingTimeout := fmt.Sprintf("%s-stopping-timeout", svc.Name)
	svc.Bus.Publish(events.Event{Code: events.Stopping, Source: svc.Name})
	if svc.stoppingEvent != events.NonEvent {
		if svc.stoppingTimeout > 0 {
			// not having this set is a programmer error not a runtime error
			events.NewEventTimeout(ctx, svc.Rx,
				svc.stoppingTimeout, stoppingTimeout)
		}
	loop:
		for {
			event := <-svc.Rx
			switch event {
			case svc.stoppingEvent:
				break loop
			case events.Event{events.Stopping, stoppingTimeout}:
				break loop
			}
		}
	}
	svc.Unsubscribe(svc.Bus)
	svc.Deregister()
	close(svc.Rx)
	cancel()
	svc.Bus.Publish(events.Event{Code: events.Stopped, Source: svc.Name})
	svc.Flush <- true
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (svc *Service) String() string {
	return "services.Service[" + svc.Name + "]" // TODO: is there a better representation???
}
