package events

// EventSubscriber is an interface for subscribers that subscribe/unsubscribe
// from the EventBus and receive Events.
type EventSubscriber interface {
	Subscribe(*EventBus)
	Unsubscribe()
	Receive(Event)
}

// Subscriber represents an object which recieves events through the Event bus
// through its receive channel.
type Subscriber struct {
	Rx  chan Event
	Bus *EventBus
}

// Subscribe subscribes a subscriber to the EventBus
func (s *Subscriber) Subscribe(bus *EventBus) {
	s.Bus = bus
	bus.Subscribe(s)
}

// Unsubscribe unsubscribes the subscriber from the EventBus.
func (s *Subscriber) Unsubscribe() {
	s.Bus.Unsubscribe(s)
}

// Receive receives an Event through the receive channel.
func (s *Subscriber) Receive(event Event) {
	s.Rx <- event
}

// Wait waits for the subscriber's EventBus to complete its wait group.
func (s *Subscriber) Wait() {
	s.Bus.done.Wait()
}
