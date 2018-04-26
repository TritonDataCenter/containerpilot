package events

import (
	"sync"
	"testing"
)

type TestPublisher struct {
	Publisher
}

func NewTestPublisher(bus *EventBus) *TestPublisher {
	tp := &TestPublisher{}
	tp.Register(bus)
	return tp
}

type TestSubscriber struct {
	results []Event
	lock    *sync.RWMutex

	Subscriber
}

func NewTestSubscriber() *TestSubscriber {
	my := &TestSubscriber{
		lock:    &sync.RWMutex{},
		results: []Event{},
	}
	my.Rx = make(chan Event, 100)
	return my
}

func (ts *TestSubscriber) Run(bus *EventBus) {
	ts.Subscribe(bus)
	go func() {
		for event := range ts.Rx {
			switch event.Code {
			case Quit:
				ts.lock.Lock()
				ts.results = append(ts.results, event)
				ts.Unsubscribe()
				close(ts.Rx)
				break
			default:
				ts.lock.Lock()
				ts.results = append(ts.results, event)
				ts.lock.Unlock()
			}
		}
	}()
}

func TestPubSubTypes(t *testing.T) {
	bus := NewEventBus()
	tp := NewTestPublisher(bus)
	defer tp.Unregister()
	ts := NewTestSubscriber()
	ts.Run(bus)

	expected := []Event{
		{Code: Startup, Source: "serviceA"},
	}
	for _, event := range expected {
		tp.Publish(event)
	}

	for i, found := range ts.results {
		if expected[i] != found {
			t.Fatalf("expected: %v\ngot: %v", expected, ts.results)
		}
	}
}
