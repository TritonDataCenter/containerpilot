package events

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestPublisher struct {
	Publisher
}

func NewTestPublisher(bus *EventBus) *TestPublisher {
	pub := &TestPublisher{}
	pub.Register(bus)
	return pub
}

type TestSubscriber struct {
	results []Event
	lock    *sync.RWMutex

	Subscriber
}

func NewTestSubscriber() *TestSubscriber {
	sub := &TestSubscriber{
		lock:    &sync.RWMutex{},
		results: []Event{},
	}
	sub.Rx = make(chan Event, 100)
	return sub
}

func (ts *TestSubscriber) Run(ctx context.Context, bus *EventBus) {
	ts.Subscribe(bus)
	go func() {
		defer func() {
			ts.Unsubscribe()
			ts.Wait()
			close(ts.Rx)
		}()
		for {
			select {
			case event, ok := <-ts.Rx:
				if !ok {
					return
				}
				ts.lock.Lock()
				ts.results = append(ts.results, event)
				ts.lock.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Plumb a basic pub/sub interaction to test out the choreography between them.
func TestPubSubInterfaces(t *testing.T) {
	bus := NewEventBus()
	tp := NewTestPublisher(bus)
	defer tp.Unregister()
	ts := NewTestSubscriber()
	ctx, cancel := context.WithCancel(context.Background())
	ts.Run(ctx, bus)

	expected := []Event{
		{Startup, "serviceA"},
	}
	for _, event := range expected {
		tp.Publish(event)
	}
	cancel()
	results := bus.DebugEvents()

	if !reflect.DeepEqual(expected, results) {
		t.Fatalf("expected: %v\ngot: %v", expected, results)
	}

	for n, found := range results {
		mesg := fmt.Sprintf("expected: %v\ngot: %v", expected, results)
		assert.Equal(t, expected[n], found, mesg)
	}
}

func TestPublishSignal(t *testing.T) {
	bus := NewEventBus()
	ts := NewTestSubscriber()
	ctx, cancel := context.WithCancel(context.Background())
	ts.Run(ctx, bus)

	signals := []string{"SIGHUP", "SIGUSR2"}
	expected := make([]Event, len(signals))
	for n, sig := range signals {
		expected[n] = Event{Code: Signal, Source: sig}
		bus.PublishSignal(sig)
	}
	cancel()
	results := bus.DebugEvents()

	if !reflect.DeepEqual(expected, results) {
		t.Fatalf("expected: %v\ngot: %v", expected, ts.results)
	}

	for n, found := range results {
		assert.Equal(t, found, expected[n])
	}
}
