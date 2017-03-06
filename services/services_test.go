package services

import (
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
)

func TestService(t *testing.T) {

	bus := events.NewEventBus()
	svc := Service{Name: "myservice"}
	svc.Rx = make(chan events.Event, 1000)
	svc.Flush = make(chan bool)
	svc.startupEvent = events.Event{Code: events.ExitSuccess, Source: "upstream"}
	svc.startupTimeout = 60
	svc.restartsRemain = 0
	svc.restartLimit = 0
	svc.heartbeat = 3

	svc.Run(bus)
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: "serviceA"})

	svc.Close()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: "serviceA"}) // should not panic
}

func TestServiceTimeout(t *testing.T) {

	bus := events.NewEventBus()
	svc := Service{Name: "myservice"}
	svc.Rx = make(chan events.Event, 1000)
	svc.Flush = make(chan bool)
	svc.startupEvent = events.Event{Code: events.Startup}
	svc.startupTimeout = 1
	svc.restartsRemain = 0
	svc.restartLimit = 0
	svc.heartbeat = 3

	svc.Run(bus)
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: "serviceA"})

	// note that we can't send a .Close() after this b/c we've timed out
	// and we'll end up blocking forever
	time.Sleep(1 * time.Second)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: "serviceA"}) // should not panic
}
