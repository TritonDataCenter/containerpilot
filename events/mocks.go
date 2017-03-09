package events

import (
	"context"
	"time"
)

// DebugSubscriber is a test helper or for instrumentation/debugging
type DebugSubscriber struct {
	EventHandler
	Results []Event
	max     int
}

func NewDebugSubscriber(bus *EventBus, max int) *DebugSubscriber {
	ds := &DebugSubscriber{
		Results: []Event{},
		max:     max,
	}
	ds.Rx = make(chan Event, 100)
	ds.Flush = make(chan bool)
	ds.Bus = bus
	return ds
}

func (ds *DebugSubscriber) Run(timeout time.Duration) {
	ds.Subscribe(ds.Bus)
	if timeout == 0 {
		timeout = time.Duration(100 * time.Millisecond)
	}
	NewEventTimeout(context.Background(), ds.Rx, timeout, "DebugSubscriberTimeout")
	go func() {
		for {
			event := <-ds.Rx
			ds.Results = append(ds.Results, event)
			if len(ds.Results) == ds.max {
				break
			}
			switch event {
			case
				Event{TimerExpired, "DebugSubscriberTimeout"},
				GlobalShutdown,
				QuitByClose:
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
