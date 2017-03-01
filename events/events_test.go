package events

import (
	"reflect"
	"sync"
	"testing"
)

func TestSafeUnsubscribe(t *testing.T) {
	bus := NewEventBus()
	ts := NewTestSubscriber(bus)
	ts.Subscribe(bus)

	ts.Run()
	bus.Publish(Event{Code: Started, Name: "serviceA"})
	ts.Close()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	bus.Publish(Event{Code: Started, Name: "serviceB"}) // should not panic

	expected := []Event{
		Event{Code: Started, Name: "serviceA"},
		Event{Code: Quit},
	}

	for _, result := range ts.results {
		if result.Name == "serviceB" {
			t.Fatal("got Event after we closed receiver")
		}
	}
	if !reflect.DeepEqual(expected, ts.results) {
		t.Fatalf("expected: %v\ngot: %v", expected, ts.results)
	}
}

/*
Dummy TestSubscriber as test helpers
*/

type TestSubscriber struct {
	EventHandler
	results []Event
	lock    *sync.RWMutex
}

func NewTestSubscriber(bus *EventBus) *TestSubscriber {
	my := &TestSubscriber{lock: &sync.RWMutex{}, results: []Event{}}
	my.Rx = make(chan Event)
	my.Flush = make(chan bool)
	my.Bus = bus
	return my
}

func (ts *TestSubscriber) Run() {
	go func() {
		for event := range ts.Rx {
			switch event.Code {
			case Quit:
				ts.lock.Lock()
				ts.results = append(ts.results, event)
				ts.Unsubscribe(ts.Bus)
				ts.Flush <- true
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
