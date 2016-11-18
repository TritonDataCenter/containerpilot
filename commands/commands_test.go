package commands

import (
	"errors"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
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
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Expected panic but did not.")
		}
	}()
	cmd, _ := NewCommand("./testdata/test.sh failStuff --debug", "0")
	if exitCode, _ := RunAndWait(cmd, nil); exitCode != 255 {
		t.Errorf("Expected exit code 255 but got %d", exitCode)
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

func TestRunWithTimeout(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh sleepStuff", "200ms")
	fields := log.Fields{"process": "test"}
	RunWithTimeout(cmd, fields)

	// Ensure the task has time to start
	runtime.Gosched()
	// Wait for task to start + 250ms
	ticker := time.NewTicker(650 * time.Millisecond)
	select {
	case <-ticker.C:
		ticker.Stop()
		err := cmd.Kill() // make sure we don't keep running
		if err == nil {
			// firing Kill should throw an error
			t.Fatalf("Command was not stopped by timeout")
		} else {
			if err.Error() != "os: process already finished" {
				t.Fatalf("Command.Kill got unexpected error: %s", err)
			}
		}
	}
}

func TestRunWithTimeoutFailed(t *testing.T) {

	log.SetLevel(log.DebugLevel)
	defer log.SetLevel(log.InfoLevel)

	tmp, _ := ioutil.TempFile("", "tmp")
	defer os.Remove(tmp.Name())

	log.SetOutput(tmp)
	defer log.SetOutput(os.Stdout)

	cmd, _ := NewCommand("./testdata/test.sh failStuff --debug", "100ms")
	fields := log.Fields{"process": "test"}
	if err := RunWithTimeout(cmd, fields); err == nil {
		t.Errorf("Expected error but got nil")
	}
	time.Sleep(200 * time.Millisecond)

	buf, _ := ioutil.ReadFile(tmp.Name())
	logs := string(buf)

	if strings.Contains(logs, "timeout after") {
		t.Fatalf("RunWithTimeout failed to cancel timeout after failure: %v", logs)
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
