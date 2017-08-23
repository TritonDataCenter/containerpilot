package events

type EventPublisher interface {
	Publish(Event)
	Register(*EventBus)
	Unregister()
}

type Publisher struct {
	Bus *EventBus
}

func (p *Publisher) Publish(event Event) {
	p.Bus.Publish(event)
}

func (p *Publisher) Register(bus *EventBus) {
	p.Bus = bus
	bus.Register(p)
}

func (p *Publisher) Unregister() {
	p.Bus.Unregister(p)
}

func (p *Publisher) Wait() {
	p.Bus.done.Wait()
}

func (p *Publisher) Quit() {
	p.Bus.done.Wait()
}
