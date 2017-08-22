package events

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// EventBus manages the state of and transmits messages to all its Subscribers
type EventBus struct {
	registry map[Subscriber]bool
	lock     *sync.RWMutex
	reload   bool
	done     sync.WaitGroup

	// circular buffer of events
	head int
	tail int
	buf  []Event
}

func (bus *EventBus) enqueue(event Event) {
	bus.buf[bus.mod(bus.head+1)] = event
	old := bus.head
	bus.head = (bus.head + 1) % len(bus.buf)
	if old != -1 && bus.head == bus.tail {
		bus.tail = bus.mod(bus.tail + 1)
	}
}

// DebugEvents ...
func (bus *EventBus) DebugEvents() []Event {
	time.Sleep(100 * time.Millisecond)
	events := []Event{}
	for {
		if bus.head == -1 {
			break
		}
		event := bus.buf[bus.mod(bus.tail)]
		if bus.tail == bus.head {
			bus.head = -1
			bus.tail = 0
		} else {
			bus.tail = bus.mod(bus.tail + 1)
		}
		if event == NonEvent {
			break
		}
		events = append(events, event)
	}
	return events
}

func (bus *EventBus) mod(p int) int {
	return p % len(bus.buf)
}

var collector *prometheus.CounterVec

func init() {
	collector = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "containerpilot_events",
		Help: "count of ContainerPilot events, partitioned by type and source",
	}, []string{"code", "source"})
	prometheus.MustRegister(collector)
}

// NewEventBus initializes an EventBus. We need this rather than a struct
// literal so that we know our channels are non-nil (which block sends).
func NewEventBus() *EventBus {
	lock := &sync.RWMutex{}
	reg := make(map[Subscriber]bool)
	buf := make([]Event, 10)
	for i := range buf {
		buf[i] = Event{}
	}
	bus := &EventBus{registry: reg, lock: lock, reload: false,
		buf: buf, head: -1, tail: 0}
	return bus
}

// Register the Publisher for all Events
func (bus *EventBus) Register(publisher EventPublisher) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	bus.done.Add(1)
}

// Unregister the Publisher from all Events
func (bus *EventBus) Unregister(publisher EventPublisher) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	bus.done.Done()
}

// Subscribe the Subscriber for all Events
func (bus *EventBus) Subscribe(subscriber EventSubscriber) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	sub := subscriber.(Subscriber)
	bus.registry[sub] = true

	// internal subscribers like the control socket and telemetry server
	// will never unregister from events, but we want to be able to exit
	// if len(isInternal) == 0 || !isInternal[0] {
	bus.done.Add(1)
	// }
}

// Unsubscribe the Subscriber from all Events
func (bus *EventBus) Unsubscribe(subscriber EventSubscriber) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	sub := subscriber.(Subscriber)
	if _, ok := bus.registry[sub]; ok {
		delete(bus.registry, sub)
	}
	// if len(isInternal) == 0 || !isInternal[0] {
	bus.done.Done()
	// }
}

// Publish an Event to all Subscribers
func (bus *EventBus) Publish(event Event) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	log.Debugf("event: %v", event)
	collector.WithLabelValues(event.Code.String(), event.Source).Inc()
	for subscriber := range bus.registry {
		// sending to an unsubscribed Subscriber shouldn't be a runtime
		// error, so this is in intentionally allowed to panic here
		subscriber.Receive(event)
	}
	bus.enqueue(event)
}

// SetReloadFlag sets the flag that Wait will use to signal to the main
// App that we want to restart rather than be shut down
func (bus *EventBus) SetReloadFlag() {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	bus.reload = true
}

// Shutdown asks all Subscribers to halt by sending the GlobalShutdown
// message. Subscribers are responsible for handling this message.
func (bus *EventBus) Shutdown() {
	bus.Publish(GlobalShutdown)
}

// Wait blocks until the EventBus registry is unpopulated. Returns true
// if the "reload" flag was set.
func (bus *EventBus) Wait() bool {
	bus.done.Wait()
	bus.lock.RLock()
	defer bus.lock.RUnlock()
	return bus.reload
}
