package events

import (
	"sync"
)

type Event struct {
	Code   EventCode
	Source string
}

// go:generate stringer -type EventCode
type EventCode int

const (
	ExitSuccess EventCode = iota
	ExitFailed
	StatusHealthy
	StatusUnhealthy
	StatusChanged
	TimerExpired
	EnterMaintenance
	ExitMaintenance
	Quit
	Startup  // fired once after events are set up and event loop is started
	Shutdown // fired once after all jobs exit or on receiving SIGTERM
)

const (
	Global = "global" // used by global event sources
	Closed = "closed" // used by Close() method only and passed directly to chan
)

type EventBus struct {
	registry map[Subscriber]bool
	lock     *sync.RWMutex
}

func NewEventBus() *EventBus {
	lock := &sync.RWMutex{}
	reg := make(map[Subscriber]bool)
	bus := &EventBus{registry: reg, lock: lock}
	return bus
}

// Register the Subscriber for all Events
func (bus *EventBus) Register(subscriber Subscriber) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	bus.registry[subscriber] = true
}

// Unregister the Subscriber from all Events
func (bus *EventBus) Unregister(subscriber Subscriber) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if _, ok := bus.registry[subscriber]; ok {
		delete(bus.registry, subscriber)
	}
}

// Publish an Event to all Subscribers
func (bus *EventBus) Publish(event Event) {
	bus.lock.RLock()
	defer bus.lock.RUnlock()
	for subscriber, _ := range bus.registry {
		// sending to an unsubscribed Subscriber shouldn't be a runtime
		// error, so this is in intentionally allowed to panic here
		subscriber.Receive(event)
	}
}

// Shutdown all Subscribers
func (bus *EventBus) Shutdown() {
	bus.lock.RLock()
	defer bus.lock.RUnlock()
	for subscriber, _ := range bus.registry {
		// sending to an unsubscribed Subscriber shouldn't be a runtime
		// error, so this is in intentionally allowed to panic here
		subscriber.Close()
	}
}
