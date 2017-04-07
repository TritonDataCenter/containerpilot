package checks

import (
	"fmt"
	"testing"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests/mocks"
)

func TestHealthCheckExecOk(t *testing.T) {
	cfg := &Config{
		Name:    "mycheckOk",
		Exec:    "./testdata/test.sh doStuff --debug",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runHealthCheckTest(cfg, 5)
	poll := events.Event{events.TimerExpired, "check.mycheckOk.poll"}
	exitOk := events.Event{events.ExitSuccess, "check.mycheckOk"}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}
}

func TestHealthCheckExecFail(t *testing.T) {
	cfg := &Config{
		Name:    "mycheckFail",
		Exec:    "./testdata/test.sh failStuff",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runHealthCheckTest(cfg, 7)
	poll := events.Event{events.TimerExpired, "check.mycheckFail.poll"}
	exitOk := events.Event{events.ExitFailed, "check.mycheckFail"}
	errMsg := events.Event{events.Error, "check.mycheckFail: exit status 255"}

	if got[exitOk] != 2 || got[poll] != 2 ||
		got[events.QuitByClose] != 1 || got[errMsg] != 2 {
		t.Fatalf("expected 2 failed poll events but got %v", got)
	}
}

func runHealthCheckTest(cfg *Config, count int) map[events.Event]int {
	bus := events.NewEventBus()
	ds := mocks.NewDebugSubscriber(bus, count)
	ds.Run(0)
	cfg.Validate()
	check := NewHealthCheck(cfg)
	check.Run(bus)

	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s.poll", cfg.Name)}
	bus.Publish(poll)
	bus.Publish(poll) // Ensure we can run it more than once
	check.Close()
	ds.Close()

	got := map[events.Event]int{}
	for _, result := range ds.Results {
		got[result]++
	}
	return got
}
