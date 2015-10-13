package main

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestArgParse(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = nil
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"this", "-poll", "20", "/root/examples/test.sh", "doStuff", "--debug"}
	parseArgs()
	if pollTime != 20 {
		t.Errorf("Expected pollTime to be 20 but got: %d", pollTime)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/root/examples/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
	}
}

// Verify we have no obvious crashing paths in the poll code and that we handle
// a closed channel immediately as expected and gracefully.
func TestPoll(t *testing.T) {
	quit := poll(func(args ...string) {
		time.Sleep(5 * time.Second)
		t.Errorf("We should never reach this code because the channel should close.")
		return
	}, "exec", "arg1")
	close(quit)
}

func TestRunSuccess(t *testing.T) {
	args := []string{"/root/examples/test.sh", "doStuff", "--debug"}
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
	args := []string{"/root/examples/test.sh", "failStuff", "--debug"}
	if exitCode, _ := run(args...); exitCode != 255 {
		t.Errorf("Expected exit code 255 but got %d", exitCode)
	}
}
