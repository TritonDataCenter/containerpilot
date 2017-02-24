package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
)

func TestRunAndWaitSuccess(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh doStuff --debug", "0")
	cmd.Name = "APP"
	if exitCode, _ := RunAndWait(cmd, nil); exitCode != 0 {
		t.Errorf("Expected exit code 0 but got %d", exitCode)
	}
	if pid := os.Getenv("CONTAINERPILOT_APP_PID"); pid == "" {
		t.Errorf("Expected CONTAINERPILOT_APP_PID to be set")
	}
}

func BenchmarkRunAndWaitSuccess(b *testing.B) {
	cmd, _ := NewCommand("./testdata/test.sh doNothing", "0")
	for i := 0; i < b.N; i++ {
		RunAndWait(cmd, nil)
	}
}

func TestRunAndWaitFailed(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh failStuff --debug", "0")
	if exitCode, _ := RunAndWait(cmd, nil); exitCode != 255 {
		t.Errorf("Expected exit code 255 but got %d", exitCode)
	}
}

func TestRunAndWaitInvalidCommand(t *testing.T) {
	cmd, _ := NewCommand("./testdata/invalidCommand", "0")
	if exitCode, _ := RunAndWait(cmd, nil); exitCode != 127 {
		t.Errorf("Expected exit code 127 but got %d", exitCode)
	}
}

func TestRunAndWaitForOutput(t *testing.T) {

	cmd, _ := NewCommand("./testdata/test.sh doStuff --debug", "0")
	if out, err := RunAndWaitForOutput(cmd); err != nil {
		t.Fatalf("Unexpected error from 'test.sh doStuff': %s", err)
	} else if out != "Running doStuff with args: --debug\n" {
		t.Fatalf("Unexpected output from 'test.sh doStuff': %s", out)
	}

	// Ensure bad commands return error
	cmd2, _ := NewCommand("./testdata/doesNotExist.sh", "0")
	if out, err := RunAndWaitForOutput(cmd2); err == nil {
		t.Fatalf("Expected error from 'doesNotExist.sh' but got %s", out)
	} else if err.Error() != "fork/exec ./testdata/doesNotExist.sh: no such file or directory" {
		t.Fatalf("Unexpected error from 'doesNotExist.sh': %s", err)
	}
}

// We want to make sure test tasks don't run forever and so if they
// exceed their timeouts and don't return an error we want to know that.
func failTestIfExceedingTimeout(t *testing.T, cmd *Command) error {
	fields := log.Fields{"process": "test"}

	c := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	go func() { c <- RunWithTimeout(cmd, fields) }()
	select {
	case <-ctx.Done():
		cmd.Kill()
		return fmt.Errorf("command was not stopped by timeout")
	case err := <-c:
		return err
	}
}

// make sure we're backwards compatible for now
func TestRunWithTimeoutZero(t *testing.T) {
	cmd, _ := NewCommand("sleep 2", "0")
	err := failTestIfExceedingTimeout(t, cmd)
	if err == nil || err.Error() != "sleep: signal: killed" {
		t.Fatalf("failed to stop command on timeout: %v", err)
	}
}

func TestRunWithTimeoutKilled(t *testing.T) {
	cmd, _ := NewCommand("sleep 2", "200ms")
	err := failTestIfExceedingTimeout(t, cmd)
	if err == nil || err.Error() != "sleep: signal: killed" {
		t.Fatalf("failed to stop command on timeout: %v", err)
	}
}

func TestRunWithTimeoutChildrenKilledToo(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh sleepStuff", "200ms")
	err := failTestIfExceedingTimeout(t, cmd)
	if err == nil || err.Error() != "./testdata/test.sh: signal: killed" {
		t.Fatalf("failed to stop command on timeout: %v", err)
	}
}

func TestRunWithTimeoutCommandFailed(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh failStuff --debug", "100ms")
	err := failTestIfExceedingTimeout(t, cmd)
	if err == nil || err.Error() != "./testdata/test.sh: exit status 255" {
		t.Fatalf("failed to stop command: %v", err)
	}
}

func TestRunWithTimeoutInvalidCommand(t *testing.T) {
	cmd, _ := NewCommand("./testdata/invalidCommand", "100ms")
	err := failTestIfExceedingTimeout(t, cmd)
	if err == nil ||
		err.Error() != "fork/exec ./testdata/invalidCommand: no such file or directory" {
		t.Errorf("Expected 'no such file' error but got %v", err)
	}
}

func TestEmptyCommand(t *testing.T) {
	if cmd, err := NewCommand("", "0"); cmd != nil || err == nil {
		t.Errorf("Expected exit (nil, err) but got %s, %s", cmd, err)
	}
}

func TestReuseCmd(t *testing.T) {
	cmd, _ := NewCommand("true", "0")
	if code, err := RunAndWait(cmd, nil); code != 0 || err != nil {
		t.Errorf("Expected exit (0,nil) but got (%d,%s)", code, err)
	}
	if code, err := RunAndWait(cmd, nil); code != 0 || err != nil {
		t.Errorf("Expected exit (0,nil) but got (%d,%s)", code, err)
	}
}

func TestGetTimeout(t *testing.T) {
	var (
		dur time.Duration
		err error
	)
	dur, err = getTimeout("1s")
	expectDuration(t, dur, time.Duration(time.Second), err, nil)

	dur, err = getTimeout("")
	expectDuration(t, dur, time.Duration(0), err, nil)

	dur, err = getTimeout("x")
	expectDuration(t, dur, time.Duration(0),
		err, errors.New("time: invalid duration x"))

	dur, err = getTimeout("0")
	expectDuration(t, dur, time.Duration(0), err, nil)

	dur, err = getTimeout("1h")
	expectDuration(t, dur, time.Duration(time.Hour), err, nil)

	// TODO: we can't really do this in the getTimeout b/c of the need
	// to support commands without timeout. In v3 we should consider
	// forcing this requirement.
	// dur, err = getTimeout("1ns")
	// expectDuration(t, dur, time.Duration(0),
	// 	err, errors.New("timeout 1ns cannot be less that 1ms"))

}

func expectDuration(t *testing.T, actual, expected time.Duration,
	err, expectedErr error) {

	if expectedErr == nil && err != nil {
		t.Fatalf("got unexpected error '%s'", err)
	}
	if expectedErr != nil && err == nil {
		t.Fatalf("did not get expected error '%s'", expectedErr)
	}
	if expectedErr != nil && err.Error() != expectedErr.Error() {
		t.Fatalf("expected error '%s' but got '%s'", expectedErr, err)
	}
	if expected != actual {
		t.Errorf("expected duration %v but got %v", expected, actual)
	}
}
