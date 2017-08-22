package events

// Subscriber is an interface for types that will receive messages
// from an EventBus
type EventSubscriber interface {
	Subscribe(*EventBus)
	Unsubscribe()
	Receive(Event)
}

type Subscriber struct {
	Rx  chan Event
	Bus *EventBus
}

func (s *Subscriber) Subscribe(bus *EventBus) {
	s.Bus = bus
	bus.Subscribe(s)
}

func (s *Subscriber) Unsubscribe() {
	s.Bus.Unsubscribe(s)
}

func (s *Subscriber) Receive(event Event) {
	s.Rx <- event
}

func (s *Subscriber) Wait() {
	s.Bus.done.Wait()
}
