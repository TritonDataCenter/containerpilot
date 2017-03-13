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

	// timing and restarts
	startupEvent   events.Event
	startupTimeout time.Duration
	heartbeat      time.Duration
	restartLimit   int
	restartsRemain int
	frequency      time.Duration

	events.EventHandler // Event handling
}

// NewService creates a new Service from a ServiceConfig
func NewService(cfg *ServiceConfig) *Service {
	service := &Service{
		Name:             cfg.Name,
		exec:             cfg.exec,
		heartbeat:        cfg.heartbeatInterval,
		discoveryService: cfg.discoveryService,
		Definition:       cfg.definition,
		startupEvent:     cfg.startupEvent,
		startupTimeout:   cfg.startupTimeout,
		restartLimit:     cfg.restartLimit,
		restartsRemain:   cfg.restartLimit,
		frequency:        cfg.freqInterval,
	}
	service.Rx = make(chan events.Event, eventBufferSize)
	service.Flush = make(chan bool)
	return service
}

// FromServiceConfigs ...
func FromServiceConfigs(cfgs []*ServiceConfig) []*Service {
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
				// TODO: we should check svc.Status here too I think
				svc.Bus.Publish(events.Event{
					Code: events.TimerExpired, Source: svc.Name})
				svc.Rx <- events.Event{Code: events.Quit, Source: svc.Name}
			case events.Event{events.TimerExpired, runEverySource}:
				if svc.restartLimit != unlimitedRestarts &&
					svc.restartsRemain <= 0 {
					break
				}
				svc.restartsRemain--
				svc.Rx <- svc.startupEvent
			case
				events.Event{events.Quit, svc.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				svc.Unsubscribe(svc.Bus)
				svc.Deregister()
				close(svc.Rx)
				cancel()
				svc.Flush <- true
				return
			case
				// TODO: check.Name won't always match svc.Name
				events.Event{events.ExitSuccess, svc.Name},
				events.Event{events.ExitFailed, svc.Name}:
				if (svc.restartLimit != unlimitedRestarts &&
					svc.restartsRemain <= 0) || svc.frequency > 0 {
					break
				}
				svc.restartsRemain--
				svc.Rx <- svc.startupEvent
			case svc.startupEvent:
				svc.StartService(ctx)
			}
		}
	}()
}
