package events

// EventSubscriber is an interface for subscribers that subscribe/unsubscribe
// from the EventBus and receive Events.
type EventSubscriber interface {
	Subscribe(*EventBus)
	Unsubscribe()
	Receive(Event)
}

// Subscriber represents an object which receives events through the Event bus
// through its receive channel.
type Subscriber struct {
	Rx  chan Event
	Bus *EventBus
}

// Subscribe subscribes a subscriber to the EventBus
func (sub *Subscriber) Subscribe(bus *EventBus) {
	sub.Bus = bus
	bus.Subscribe(sub)
}

// Unsubscribe unsubscribes the subscriber from the EventBus.
func (sub *Subscriber) Unsubscribe() {
	sub.Bus.Unsubscribe(sub)
}

// Receive receives an Event through the receive channel.
func (sub *Subscriber) Receive(event Event) {
	sub.Rx <- event
}

// Wait waits for the subscriber's EventBus to complete its wait group.
func (sub *Subscriber) Wait() {
	sub.Bus.done.Wait()
}
