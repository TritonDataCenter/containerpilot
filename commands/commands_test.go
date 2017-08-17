package commands

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCommandRunWithTimeoutZero(t *testing.T) {
	cmd, _ := NewCommand("sleep 2", time.Duration(0), nil)
	got := runtestCommandRun(cmd)
	timedout := events.Event{events.ExitFailed, "sleep"}
	if got[timedout] != 1 {
		t.Fatalf("stopped command prior to test timeout, got events %v", got)
	}
}

func TestCommandRunWithTimeoutKilled(t *testing.T) {
	cmd, _ := NewCommand("sleep 2", time.Duration(100*time.Millisecond), nil)
	cmd.Name = t.Name()
	got := runtestCommandRun(cmd)
	testTimeout := events.Event{events.TimerExpired, "DebugSubscriberTimeout"}
	expired := events.Event{events.ExitFailed, t.Name()}
	errMsg := events.Event{events.Error, fmt.Sprintf("%s: signal: killed", cmd.Name)}
	if got[testTimeout] > 0 || got[expired] != 1 || got[errMsg] != 1 {
		t.Fatalf("expected:\n%v\n%v\ngot events:\n%v", expired, errMsg, got)
	}
}

func TestCommandRunChildrenKilled(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh sleepStuff",
		time.Duration(100*time.Millisecond), nil)
	cmd.Name = t.Name()
	got := runtestCommandRun(cmd)
	testTimeout := events.Event{events.TimerExpired, "DebugSubscriberTimeout"}
	expired := events.Event{events.ExitFailed, t.Name()}
	errMsg := events.Event{events.Error, fmt.Sprintf("%s: signal: killed", cmd.Name)}
	if got[testTimeout] > 0 || got[expired] != 1 || got[errMsg] != 1 {
		t.Fatalf("expected:\n%v\n%v\ngot events:\n%v", expired, errMsg, got)
	}
}

func TestCommandRunExecFailed(t *testing.T) {
	cmd, _ := NewCommand("./testdata/test.sh failStuff --debug", time.Duration(0), nil)
	got := runtestCommandRun(cmd)
	failed := events.Event{events.ExitFailed, "./testdata/test.sh"}
	errMsg := events.Event{events.Error, "./testdata/test.sh: exit status 255"}
	if got[failed] != 1 || got[errMsg] != 1 {
		t.Fatalf("expected:\n%v\n%v\ngot events:\n%v", failed, errMsg, got)
	}
}

func TestCommandRunExecInvalid(t *testing.T) {
	cmd, _ := NewCommand("./testdata/invalidCommand", time.Duration(0), nil)
	got := runtestCommandRun(cmd)
	failed := events.Event{events.ExitFailed, "./testdata/invalidCommand"}
	errMsg := events.Event{events.Error,
		"fork/exec ./testdata/invalidCommand: no such file or directory"}
	if got[failed] != 1 || got[errMsg] != 1 {
		t.Fatalf("expected:\n%v\n%v\ngot events:\n%v", failed, errMsg, got)
	}
}

func TestEmptyCommand(t *testing.T) {
	if cmd, err := NewCommand("", time.Duration(0), nil); cmd != nil || err == nil {
		t.Errorf("Expected exit (nil, err) but got %v, %s", cmd, err)
	}
}

func TestCommandRunReuseCmd(t *testing.T) {
	cmd, _ := NewCommand("true", time.Duration(0), nil)
	runtestCommandRun(cmd)
	runtestCommandRun(cmd)
}

func TestCommandPassthru(t *testing.T) {
	cmd, _ := NewCommand("true", time.Duration(0), nil)
	runtestCommandRun(cmd)
	assert.Equal(t, cmd.Cmd.Stdout, os.Stdout)

	cmd, _ = NewCommand("true", time.Duration(0), log.Fields{"job": "trueDat"})
	runtestCommandRun(cmd)
	assert.NotEqual(t, cmd.Cmd.Stdout, os.Stdout)
}

// test helpers

func runtestCommandRun(cmd *Command) map[events.Event]int {
	bus := events.NewEventBus()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	cmd.Run(ctx, bus)
	time.Sleep(300 * time.Millisecond)
	defer cancel()
	bus.Wait()
	results := bus.DebugEvents()
	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}
	return got
}
