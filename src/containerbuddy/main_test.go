package main

import (
	"testing"
	"time"
)

// Verify we have no obvious crashing paths in the poll code and that we handle
// a closed channel immediately as expected and gracefully.
func TestPoll(t *testing.T) {
	service := &ServiceConfig{Poll: 1}
	quit := poll(service, func(service Pollable, args ...string) {
		time.Sleep(5 * time.Second)
		t.Errorf("We should never reach this code because the channel should close.")
		return
	}, "exec", "arg1")
	close(quit)
}

func TestRunSuccess(t *testing.T) {
	args := []string{"/root/examples/test/test.sh", "doStuff", "--debug"}
	if exitCode, _ := run(args...); exitCode != 0 {
		t.Errorf("Expected exit code 0 but got %d", exitCode)
	}
}

func TestRunFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected panic but did not.")
		}
	}()
	args := []string{"/root/examples/test/test.sh", "failStuff", "--debug"}
	if exitCode, _ := run(args...); exitCode != 255 {
		t.Errorf("Expected exit code 255 but got %d", exitCode)
	}
}
