package watches

import (
	"context"
	"fmt"
	"testing"

	"github.com/tritondatacenter/containerpilot/discovery"
	"github.com/tritondatacenter/containerpilot/events"
	"github.com/tritondatacenter/containerpilot/tests/mocks"
)

func TestWatchPollOk(t *testing.T) {
	cfg := &Config{
		Name: "mywatchOk",
		Poll: 1,
	}
	// this discovery backend will always return true when we check
	// it for changed
	got := runWatchTest(cfg, 5, &mocks.NoopDiscoveryBackend{Val: true})
	changed := events.Event{Code: events.StatusChanged, Source: "watch.mywatchOk"}
	healthy := events.Event{Code: events.StatusHealthy, Source: "watch.mywatchOk"}
	if got[changed] != 1 || got[healthy] != 1 {
		t.Fatalf("expected 2 successful StatusHealthy events but got %v", got)
	}
}

func TestWatchPollFail(t *testing.T) {
	cfg := &Config{
		Name: "mywatchFail",
		Poll: 1,
	}
	got := runWatchTest(cfg, 3, &mocks.NoopDiscoveryBackend{Val: false})
	changed := events.Event{Code: events.StatusChanged, Source: "watch.mywatchFail"}
	unhealthy := events.Event{Code: events.StatusUnhealthy, Source: "watch.mywatchFail"}
	if got[changed] != 0 || got[unhealthy] != 0 {
		t.Fatalf("expected 2 failed poll events without changes, but got %v", got)
	}
}

func runWatchTest(cfg *Config, count int, disc discovery.Backend) map[events.Event]int {
	bus := events.NewEventBus()
	cfg.Validate(disc)
	watch := NewWatch(cfg)
	ctx := context.Background()
	watch.Run(ctx, bus)
	poll := events.Event{Code: events.TimerExpired, Source: fmt.Sprintf("%s.poll", cfg.Name)}
	watch.Receive(poll)
	watch.Receive(poll) // Ensure we can run it more than once
	watch.Receive(events.QuitByTest)
	bus.Wait()
	results := bus.DebugEvents()

	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}
	return got
}
