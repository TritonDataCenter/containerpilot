package events

import "errors"

// EventHandler should be embedded in all Runners so that we can reuse
// the code for registering and unregistering handlers. This is why the
// various fields are (unfortunately) public and we can't use struct
// literals for constructors. All NewRunner functions will need to set
// these fields explicitly:
//   runner.Rx = make(chan Event, n)
//   runner.Flush = make(chan bool)
//   runner.Bus = &EventBus{}
type EventHandler struct {
	Bus   *EventBus
	Rx    chan Event // typically buffered
	Flush chan bool  // must be unbuffered
}

// Subscribe adds the EventHandler to the list of handlers that
// receive all messages from the EventBus.
func (evh *EventHandler) Subscribe(bus *EventBus) {
	bus.Register(evh)
}

// Unsubscribe removes the EventHandler from the list of handlers
// that receive messages from the EventBus.
func (evh *EventHandler) Unsubscribe(bus *EventBus) {
	bus.Unregister(evh)
}

// Receive accepts an Event for the EventHandler's receive channel.
// Embedding struct should use a non-blocking buffered channel but
// this may be blocking in tests.
func (evh *EventHandler) Receive(e Event) {
	// TODO: instrument receives so we can report event throughput
	// statistics via Prometheus
	evh.Rx <- e
}

// Close sends a Quit message to the EventHandler and then synchronously
// waits for the EventHandler to be unregistered from all events.
func (evh *EventHandler) Close() (err error) {
	// we're going to recover from a panic here because otherwise
	// its only safe to call Close once and we have no way of
	// formalizing that except by being very careful
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("sent Close to closed handler")
		}
	}()

	evh.Rx <- QuitByClose
	<-evh.Flush
	close(evh.Flush)
	return err
}
