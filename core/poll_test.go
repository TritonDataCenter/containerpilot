package core

import (
	"testing"
	"time"
)

type DummyPollable struct{}

func (p DummyPollable) PollTime() time.Duration { return time.Duration(1) * time.Second }
func (p DummyPollable) PollAction() {
	time.Sleep(5 * time.Second)
	panic("We should never reach this code because the channel should close.")
}
func (p DummyPollable) PollStop() {}

// Verify we have no obvious crashing paths in the poll code and that we handle
// a closed channel immediately as expected and gracefully.
func TestPoll(t *testing.T) {
	app := EmptyApp()
	service := &DummyPollable{}
	quit := app.poll(service)
	close(quit)
}
