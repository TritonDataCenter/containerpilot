package events

type Runner interface {
	Run(*EventBus)

	// implementation should get these by embedding EventHandler
	Subscribe(*EventBus)
	Unsubscribe(*EventBus)
	Receive(Event)
	Close() error
}

type Subscriber interface {
	Subscribe(*EventBus)
	Unsubscribe(*EventBus)
	Receive(Event)
	Close() error
}
