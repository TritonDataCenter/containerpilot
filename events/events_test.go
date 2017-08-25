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
	bus.Publish(Event{Code: Startup, Source: "serviceA"})
	ts.Quit()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	bus.Publish(Event{Code: Startup, Source: "serviceB"}) // should not panic

	expected := []Event{
		{Code: Startup, Source: "serviceA"},
		QuitByClose,
	}

	for _, result := range ts.results {
		if result.Source == "serviceB" {
			t.Fatal("got Event after we closed receiver")
		}
	}
	if !reflect.DeepEqual(expected, ts.results) {
		t.Fatalf("expected: %v\ngot: %v", expected, ts.results)
	}
}

/*
Dummy TestSubscriber as test helpers; need this because we
don't want a circular reference with the mocks package
*/

type TestSubscriber struct {
	EventHandler
	results []Event
	lock    *sync.RWMutex
}

func NewTestSubscriber(bus *EventBus) *TestSubscriber {
	my := &TestSubscriber{lock: &sync.RWMutex{}, results: []Event{}}
	my.InitRx()
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
