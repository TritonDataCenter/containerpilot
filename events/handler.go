package events

import "sync"

// EventHandler should be embedded in all task runners so that we can
// reuse the code for registering and unregistering handlers. This is why
// the various fields are (unfortunately) public and we can't use struct
// literals for constructors. All task runner constructors will need to set
// these fields explicitly:
//   runner.Rx = make(chan Event)
//   runner.Bus = &EventBus{}
type EventHandler struct {
	Bus *EventBus
	Rx  chan Event
	wg  sync.WaitGroup
}

// Subscribe adds the EventHandler to the list of handlers that
// receive all messages from the EventBus.
func (evh *EventHandler) Subscribe(bus *EventBus, isInternal ...bool) {
	evh.wg.Add(1)
	bus.Register(evh, isInternal...)
	evh.Bus = bus
}

// Unsubscribe removes the EventHandler from the list of handlers
// that receive messages from the EventBus.
func (evh *EventHandler) Unsubscribe(bus *EventBus, isInternal ...bool) {
	evh.wg.Done()
	bus.Unregister(evh, isInternal...)
}

// Receive accepts an Event for the EventHandler's receive channel.
// Embedding struct should use a non-blocking buffered channel but
// this may be blocking in tests.
func (evh *EventHandler) Receive(e Event) {
	// TODO v3: instrument receives so we can report event throughput
	// statistics via Prometheus
	evh.Rx <- e
}

// Quit sends a Quit message to the EventHandler and then synchronously
// waits for the EventHandler to complete all in-flight work.
func (evh *EventHandler) Quit() {
	evh.Rx <- QuitByClose
	evh.wg.Wait()
}
