package mocks

import (
	"context"
	"time"

	"github.com/joyent/containerpilot/events"
)

// DebugSubscriber is a test helper or for instrumentation/debugging
type DebugSubscriber struct {
	events.EventHandler
	Results []events.Event
	max     int
}

// NewDebugSubscriber ...
func NewDebugSubscriber(bus *events.EventBus, max int) *DebugSubscriber {
	ds := &DebugSubscriber{
		Results: []events.Event{},
		max:     max,
	}
	ds.Rx = make(chan events.Event, 100)
	ds.Flush = make(chan bool)
	ds.Bus = bus
	return ds
}

// Run ...
func (ds *DebugSubscriber) Run(timeout time.Duration) {
	ds.Subscribe(ds.Bus)
	if timeout == 0 {
		timeout = time.Duration(100 * time.Millisecond)
	}
	selfTimeout := events.Event{events.TimerExpired, "DebugSubscriberTimeout"}
	events.NewEventTimeout(context.Background(), ds.Rx, timeout, "DebugSubscriberTimeout")
	go func() {
		for {
			event := <-ds.Rx
			// we don't want the mock to record its own timeouts
			if event != selfTimeout {
				ds.Results = append(ds.Results, event)
			}
			if len(ds.Results) == ds.max {
				break
			}
			switch event {
			case
				selfTimeout,
				events.GlobalShutdown,
				events.QuitByClose:
				break
			}
		}
		ds.cleanup()
	}()
}

func (ds *DebugSubscriber) cleanup() {
	ds.Unsubscribe(ds.Bus)
	close(ds.Rx)
	ds.Flush <- true
}
