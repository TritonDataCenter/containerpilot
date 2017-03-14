package watches

import (
	"fmt"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
)

func TestWatchExecOk(t *testing.T) {
	log.SetLevel(log.WarnLevel) // suppress test noise
	cfg := &Config{
		Name:    "mywatchOk",
		Exec:    "./testdata/test.sh doStuff --debug",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runWatchTest(cfg, 5)
	poll := events.Event{events.TimerExpired, "mywatchOk-watch-poll"}
	exitOk := events.Event{events.ExitSuccess, "mywatchOk"}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}
}

func TestWatchExecFail(t *testing.T) {
	log.SetLevel(log.WarnLevel) // suppress test noise
	cfg := &Config{
		Name:    "mywatchFail",
		Exec:    "./testdata/test.sh failStuff",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runWatchTest(cfg, 7)
	poll := events.Event{events.TimerExpired, "mywatchFail-watch-poll"}
	exitOk := events.Event{events.ExitFailed, "mywatchFail"}
	errMsg := events.Event{events.Error, "mywatchFail: exit status 255"}
	if got[exitOk] != 2 || got[poll] != 2 ||
		got[events.QuitByClose] != 1 || got[errMsg] != 2 {
		t.Fatalf("expected 2 failed poll events but got %v", got)
	}
}

func runWatchTest(cfg *Config, count int) map[events.Event]int {
	bus := events.NewEventBus()
	ds := events.NewDebugSubscriber(bus, count)
	ds.Run(0)
	cfg.Validate(&NoopServiceBackend{})
	watch := NewWatch(cfg)
	watch.Run(bus)

	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-watch-poll", cfg.Name)}
	bus.Publish(poll)
	bus.Publish(poll) // Ensure we can run it more than once
	watch.Close()
	ds.Close()

	got := map[events.Event]int{}
	for _, result := range ds.Results {
		got[result]++
	}
	return got
}

// Mock Discovery
// TODO this should probably go into the discovery package for use in testing everywhere
type NoopServiceBackend struct{}

func (c *NoopServiceBackend) SendHeartbeat(service *discovery.ServiceDefinition)      { return }
func (c *NoopServiceBackend) CheckForUpstreamChanges(backend, tag string) bool        { return true }
func (c *NoopServiceBackend) MarkForMaintenance(service *discovery.ServiceDefinition) {}
func (c *NoopServiceBackend) Deregister(service *discovery.ServiceDefinition)         {}
