package services

import (
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
)

func TestServiceEvents(t *testing.T) {

	svc := ServiceEvents{ID: "myservice"}
	svc.Bus = events.NewEventBus()
	svc.Rx = make(chan events.Event, 1000)
	svc.Flush = make(chan bool)
	svc.startupEvent = events.Event{Code: events.ExitSuccess, Source: "upstream"}
	svc.startupTimeout = 60
	svc.restarts = 0
	svc.heartbeat = 3

	svc.Run()
	svc.Bus.Publish(events.Event{Code: events.Started, Name: "serviceA"})

	svc.Close()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.Event{Code: events.Started, Name: "serviceA"}) // should not panic
}

func TestServiceTimeout(t *testing.T) {

	svc := ServiceEvents{ID: "myservice"}
	svc.Bus = events.NewEventBus()
	svc.Rx = make(chan events.Event, 1000)
	svc.Flush = make(chan bool)
	svc.startupEvent = events.Event{Code: events.Startup}
	svc.startupTimeout = 1
	svc.restarts = 0
	svc.heartbeat = 3

	svc.Run()
	svc.Bus.Publish(events.Event{Code: events.Started, Name: "serviceA"})

	// note that we can't send a .Close() after this b/c we've timed out
	// and we'll end up blocking forever
	time.Sleep(1 * time.Second)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.Event{Code: events.Started, Name: "serviceA"}) // should not panic
}
