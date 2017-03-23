package events

// Subscriber is an interface for types that will receive messages
// from an EventBus
type Subscriber interface {
	Subscribe(*EventBus)
	Unsubscribe(*EventBus)
	Receive(Event)
	Close() error
}
