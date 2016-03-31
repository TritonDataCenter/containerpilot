package core

import (
	"testing"
	"time"
)

type DummyPollable struct{}

func (p DummyPollable) PollTime() int { return 1 }
func (p DummyPollable) PollAction() {
	time.Sleep(5 * time.Second)
	panic("We should never reach this code because the channel should close.")
}

// Verify we have no obvious crashing paths in the poll code and that we handle
// a closed channel immediately as expected and gracefully.
func TestPoll(t *testing.T) {
	service := &DummyPollable{}
	quit := poll(service)
	close(quit)
}
