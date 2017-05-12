package watches

import (
	"fmt"
	"testing"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests/mocks"
)

func TestWatchPollOk(t *testing.T) {
	cfg := &Config{
		Name: "mywatchOk",
		Poll: 1,
	}
	// this discovery backend will always return true when we check
	// it for changed
	got := runWatchTest(cfg, 5, &mocks.NoopDiscoveryBackend{Val: true})
	poll := events.Event{events.TimerExpired, "watch.mywatchOk.poll"}
	changed := events.Event{events.StatusChanged, "watch.mywatchOk"}
	healthy := events.Event{events.StatusHealthy, "watch.mywatchOk"}
	if got[changed] != 1 || got[poll] != 2 || got[healthy] != 1 {
		t.Fatalf("expected 2 successful StatusHealthy events but got %v", got)
	}
}

func TestWatchPollFail(t *testing.T) {
	cfg := &Config{
		Name: "mywatchFail",
		Poll: 1,
	}
	got := runWatchTest(cfg, 3, &mocks.NoopDiscoveryBackend{Val: false})
	poll := events.Event{events.TimerExpired, "watch.mywatchFail.poll"}
	changed := events.Event{events.StatusChanged, "watch.mywatchFail"}
	unhealthy := events.Event{events.StatusUnhealthy, "watch.mywatchFail"}
	if got[changed] != 0 || got[poll] != 2 || got[unhealthy] != 0 {
		t.Fatalf("expected 2 failed poll events without changes, but got %v", got)
	}
}

func runWatchTest(cfg *Config, count int, disc discovery.Backend) map[events.Event]int {
	bus := events.NewEventBus()
	cfg.Validate(disc)
	watch := NewWatch(cfg)
	watch.Run(bus)

	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s.poll", cfg.Name)}
	bus.Publish(poll)
	bus.Publish(poll) // Ensure we can run it more than once
	watch.Quit()
	bus.Wait()
	results := bus.DebugEvents()

	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}
	return got
}
