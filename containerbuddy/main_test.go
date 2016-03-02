package containerbuddy

import (
	"testing"
	"time"
)

// Verify we have no obvious crashing paths in the poll code and that we handle
// a closed channel immediately as expected and gracefully.
func TestPoll(t *testing.T) {
	service := &ServiceConfig{Poll: 1}
	quit := poll(service, func(service Pollable) {
		time.Sleep(5 * time.Second)
		t.Errorf("We should never reach this code because the channel should close.")
		return
	})
	close(quit)
}

func TestRunSuccess(t *testing.T) {
	cmd1 := strToCmd("./testdata/test.sh doStuff --debug")
	if exitCode, _ := run(cmd1); exitCode != 0 {
		t.Errorf("Expected exit code 0 but got %d", exitCode)
	}
	cmd2 := argsToCmd([]string{"./testdata/test.sh", "doStuff", "--debug"})
	if exitCode, _ := run(cmd2); exitCode != 0 {
		t.Errorf("Expected exit code 0 but got %d", exitCode)
	}
}

func TestRunFailed(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected panic but did not.")
		}
	}()
	cmd := strToCmd("./testdata/test.sh failStuff --debug")
	if exitCode, _ := run(cmd); exitCode != 255 {
		t.Errorf("Expected exit code 255 but got %d", exitCode)
	}
}

func TestRunNothing(t *testing.T) {
	if code, err := run(strToCmd("")); code != 0 || err != nil {
		t.Errorf("Expected exit (0,nil) but got (%d,%s)", code, err)
	}
}
