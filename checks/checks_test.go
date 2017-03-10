package checks

import (
	"fmt"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
)

func TestHealthCheckExecOk(t *testing.T) {
	log.SetLevel(log.WarnLevel) // suppress test noise
	cfg := &HealthCheckConfig{
		Name:    "mycheckOk",
		Exec:    "./testdata/test.sh doStuff --debug",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runHealthCheckTest(cfg, 5)
	poll := events.Event{events.TimerExpired, "mycheckOk-poll"}
	exitOk := events.Event{events.ExitSuccess, "mycheckOk"}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}
}

func TestHealthCheckExecFail(t *testing.T) {
	log.SetLevel(log.WarnLevel) // suppress test noise
	cfg := &HealthCheckConfig{
		Name:    "mycheckFail",
		Exec:    "./testdata/test.sh failStuff",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runHealthCheckTest(cfg, 7)
	poll := events.Event{events.TimerExpired, "mycheckFail-poll"}
	exitOk := events.Event{events.ExitFailed, "mycheckFail"}
	errMsg := events.Event{events.Error, "mycheckFail: exit status 255"}

	if got[exitOk] != 2 || got[poll] != 2 ||
		got[events.QuitByClose] != 1 || got[errMsg] != 2 {
		t.Fatalf("expected 2 failed poll events but got %v", got)
	}
}

func runHealthCheckTest(cfg *HealthCheckConfig, count int) map[events.Event]int {
	bus := events.NewEventBus()
	ds := events.NewDebugSubscriber(bus, count)
	ds.Run(0)
	cfg.Validate()
	check, _ := NewHealthCheck(cfg)
	check.Run(bus)

	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-poll", cfg.Name)}
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
