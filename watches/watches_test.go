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
	cfg := &WatchConfig{
		Name:    "mywatchOk",
		Exec:    "./testdata/test.sh doStuff --debug",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runWatchTest(cfg)
	poll := events.Event{events.TimerExpired, "mywatchOk-watch-poll"}
	exitOk := events.Event{events.ExitSuccess, "mywatchOk"}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}
}

func TestWatchExecFail(t *testing.T) {
	log.SetLevel(log.WarnLevel) // suppress test noise
	cfg := &WatchConfig{
		Name:    "mywatchFail",
		Exec:    "./testdata/test.sh failStuff",
		Timeout: "100ms",
		Poll:    1,
	}
	got := runWatchTest(cfg)
	poll := events.Event{events.TimerExpired, "mywatchFail-watch-poll"}
	exitOk := events.Event{events.ExitFailed, "mywatchFail"}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 failed poll events but got %v", got)
	}
}

func runWatchTest(cfg *WatchConfig) map[events.Event]int {
	bus := events.NewEventBus()
	ds := events.NewDebugSubscriber(bus, 5)
	ds.Run(0)
	cfg.Validate(&NoopServiceBackend{})
	watch, _ := NewWatch(cfg)
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
