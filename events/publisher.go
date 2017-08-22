package events

// EventPublisher is an interface for publishers that register/unregister from
// the EventBus and publish Events.
type EventPublisher interface {
	Publish(Event)
	Register(*EventBus)
	Unregister()
}

// Publisher represents an object with a Bus that implements the EventPublisher
// interface.
type Publisher struct {
	Bus *EventBus
}

// Publish publishes an Event across the Publisher's EventBus
func (pub *Publisher) Publish(event Event) {
	pub.Bus.Publish(event)
}

// Register registers the Publisher with the EventBus.
func (pub *Publisher) Register(bus *EventBus) {
	pub.Bus = bus
	bus.Register(pub)
}

// Unregister unregisters the Publisher from the EventBus.
func (pub *Publisher) Unregister() {
	pub.Bus.Unregister(pub)
}

// Wait blocks for the EventBus wait group counter to count down to zero.
func (pub *Publisher) Wait() {
	pub.Bus.done.Wait()
}
