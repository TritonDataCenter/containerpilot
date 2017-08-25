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
func (p *Publisher) Publish(event Event) {
	p.Bus.Publish(event)
}

// Register registers the Publisher with the EventBus.
func (p *Publisher) Register(bus *EventBus) {
	p.Bus = bus
	bus.Register(p)
}

// Unregister unregisters the Publisher from the EventBus.
func (p *Publisher) Unregister() {
	p.Bus.Unregister(p)
}

// Wait blocks for the EventBus wait group counter to count down to zero.
func (p *Publisher) Wait() {
	p.Bus.done.Wait()
}

// Quit blocks for the EventBus wait group counter to count down to zero.
func (p *Publisher) Quit() {
	p.Bus.done.Wait()
}
