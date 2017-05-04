package events

import (
	"sync"

	log "github.com/Sirupsen/logrus"
)

// EventBus manages the state of and transmits messages to all its Subscribers
type EventBus struct {
	registry map[Subscriber]bool
	lock     *sync.RWMutex
	reload   bool
	done     chan bool
}

// NewEventBus initializes an EventBus. We need this rather than a struct
// literal so that we know our channels are non-nil (which block sends).
func NewEventBus() *EventBus {
	lock := &sync.RWMutex{}
	reg := make(map[Subscriber]bool)
	done := make(chan bool, 1)
	bus := &EventBus{registry: reg, lock: lock, done: done, reload: false}
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
	// we want to shut down once everything has exited except for
	// the control server
	if len(bus.registry) <= 1 {
		bus.done <- true
	}
}

// Publish an Event to all Subscribers
func (bus *EventBus) Publish(event Event) {
	log.Debugf("event: %v", event)
	bus.lock.RLock()
	defer bus.lock.RUnlock()
	for subscriber := range bus.registry {
		// sending to an unsubscribed Subscriber shouldn't be a runtime
		// error, so this is in intentionally allowed to panic here
		subscriber.Receive(event)
	}
}

// SetReloadFlag sets the flag that Wait will use to signal to the main
// App that we want to restart rather than be shut down
func (bus *EventBus) SetReloadFlag() {
	bus.lock.Lock()
	bus.reload = true
	bus.lock.Unlock()
}

// Shutdown asks all Subscribers to halt by sending the GlobalShutdown
// message. Subscribers are responsible for handling this message.
func (bus *EventBus) Shutdown() {
	bus.Publish(GlobalShutdown)
}

// Wait blocks until the EventBus registry is unpopulated. Returns true
// if the "reload" flag was set.
func (bus *EventBus) Wait() bool {
	<-bus.done
	close(bus.done)
	bus.lock.RLock()
	defer bus.lock.RUnlock()
	return bus.reload
}
