package core

import (
	"context"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/tritondatacenter/containerpilot/events"
	"github.com/tritondatacenter/containerpilot/jobs"
	"github.com/tritondatacenter/containerpilot/tests/mocks"
	"github.com/stretchr/testify/assert"
)

// ------------------------------------------
// Test setup

func getSignalTestConfig() *App {
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

func getSignalEventTestConfig(signals []string) *App {
	appJobs := make([]*jobs.Job, len(signals))
	for n, sig := range signals {
		cfg := &jobs.Config{
			Name:       "test-" + sig,
			Port:       1,
			Interfaces: []string{"inet"},
			Exec:       []string{"./testdata/test.sh", "interruptSleep"},
			When:       &jobs.WhenConfig{Source: sig},
		}
		cfg.Validate(&mocks.NoopDiscoveryBackend{})
		appJobs[n] = jobs.NewJob(cfg)
	}
	app := EmptyApp()
	app.StopTimeout = 1
	app.Jobs = appJobs
	app.Bus = events.NewEventBus()
	return app
}

// Test handler for SIGTERM. Note that the SIGCHLD handler is fired
// by this same test, but that we don't have a separate unit test
// because they'll interfere with each other's state.
func TestTerminateSignal(t *testing.T) {
	stopCh := make(chan struct{}, 1)
	app := getSignalTestConfig()
	bus := app.Bus
	ctx, cancel := context.WithCancel(context.Background())
	for _, job := range app.Jobs {
		job.Subscribe(bus)
		job.Register(bus)
	}
	for _, job := range app.Jobs {
		job.Run(ctx, stopCh)
	}
	app.Terminate()
	cancel()
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

// Test handler for handling signal events SIGHUP (and SIGUSR2). Note that the
// SIGUSR1 is currently setup to handle reloading ContainerPilot's log file.
func TestSignalEvent(t *testing.T) {
	stopCh := make(chan struct{}, 1)
	signals := []string{"SIGHUP", "SIGUSR2"}
	app := getSignalEventTestConfig(signals)
	bus := app.Bus
	ctx, cancel := context.WithCancel(context.Background())
	for _, job := range app.Jobs {
		job.Subscribe(bus)
		job.Register(bus)
	}
	for _, job := range app.Jobs {
		job.Run(ctx, stopCh)
	}
	for _, sig := range signals {
		app.SignalEvent(sig)
	}

	cancel()
	bus.Wait()
	results := bus.DebugEvents()

	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}

	if !reflect.DeepEqual(got, map[events.Event]int{
		{Code: events.Signal, Source: "SIGHUP"}:         1,
		{Code: events.Signal, Source: "SIGUSR2"}:        1,
		{Code: events.Stopped, Source: "test-SIGHUP"}:   1,
		{Code: events.Stopping, Source: "test-SIGHUP"}:  1,
		{Code: events.Stopped, Source: "test-SIGUSR2"}:  1,
		{Code: events.Stopping, Source: "test-SIGUSR2"}: 1,
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

func TestToString(t *testing.T) {
	tests := []struct {
		name   string
		input  os.Signal
		output string
	}{
		{"SIGHUP", syscall.SIGHUP, "SIGHUP"},
		{"SIGUSR2", syscall.SIGUSR2, "SIGUSR2"},
		{"SIGTERM", syscall.SIGTERM, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, toString(test.input), test.output)
		})
	}
}
