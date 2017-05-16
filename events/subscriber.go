package events

// Subscriber is an interface for types that will receive messages
// from an EventBus
type Subscriber interface {
	Subscribe(*EventBus, ...bool)
	Unsubscribe(*EventBus, ...bool)
	Receive(Event)
	Quit()
}
