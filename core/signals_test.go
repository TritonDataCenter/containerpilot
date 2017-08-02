package core

import (
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/jobs"
	"github.com/joyent/containerpilot/tests/mocks"
)

// ------------------------------------------
// Test setup

func getSignalTestConfig(t *testing.T) *App {

	cfg := &jobs.Config{
		Name:       "test-service",
		Port:       1,
		Interfaces: []string{"inet"},
		Exec:       []string{"./testdata/test.sh", "interruptSleep"},
		Health: &jobs.HealthConfig{
			Heartbeat: 1,
			TTL:       1,
		},
	}
	cfg.Validate(&mocks.NoopDiscoveryBackend{})
	job := jobs.NewJob(cfg)
	app := EmptyApp()
	app.StopTimeout = 1
	app.Jobs = []*jobs.Job{job}
	app.Bus = events.NewEventBus()
	return app
}

// Test handler for SIGTERM. Note that the SIGCHLD handler is fired
// by this same test, but that we don't have a separate unit test
// because they'll interfere with each other's state.
func TestTerminateSignal(t *testing.T) {
	app := getSignalTestConfig(t)
	bus := app.Bus
	app.Jobs[0].Subscribe(bus)
	app.Jobs[0].Run()

	app.Terminate()
	bus.Wait()
	results := bus.DebugEvents()
	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}
	if !reflect.DeepEqual(got, map[events.Event]int{
		events.GlobalShutdown:                           1,
		{Code: events.Stopping, Source: "test-service"}: 1,
		{Code: events.Stopped, Source: "test-service"}:  1,
	}) {
		t.Fatalf("expected shutdown but got:\n%v", results)
	}
}

// Test that only ensures that we cover a straight-line run through
// the handleSignals setup code
func TestSignalWiring(t *testing.T) {
	app := EmptyApp()
	app.Bus = events.NewEventBus()
	app.handleSignals()
	sendAndWaitForSignal(t, syscall.SIGTERM)
}

// Helper to ensure the signal that we send has been received so that
// we don't muddy subsequent tests of the signal handler.
func sendAndWaitForSignal(t *testing.T, s os.Signal) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, s)
	me, _ := os.FindProcess(os.Getpid())
	if err := me.Signal(s); err != nil {
		t.Fatalf("Got error on %s: %v\n", s.String(), err)
	}
	select {
	case recv := <-sig:
		if recv != s {
			t.Fatalf("Expected %v but got %v\n", s, recv)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for %v\n", s)
	}
}
