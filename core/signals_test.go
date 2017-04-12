package core

import (
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/joyent/containerpilot/discovery/consul"
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
	app.StopTimeout = 5
	app.Jobs = []*jobs.Job{job}
	app.Bus = events.NewEventBus()
	return app
}

// ------------------------------------------

// Test handler for SIGUSR1
func TestMaintenanceSignal(t *testing.T) {
	app := getSignalTestConfig(t)
	if app.InMaintenanceMode() {
		t.Fatal("Should not be in maintenance mode by default")
	}

	app.ToggleMaintenanceMode()
	if !app.InMaintenanceMode() {
		t.Fatal("Should be in maintenance mode after receiving SIGUSR1")
	}

	app.ToggleMaintenanceMode()
	if app.InMaintenanceMode() {
		t.Fatal("Should not be in maintenance mode after receiving second SIGUSR1")
	}
}

// Test handler for SIGTERM. Note that the SIGCHLD handler is fired
// by this same test, but that we don't have a separate unit test
// because they'll interfere with each other's state.
func TestTerminateSignal(t *testing.T) {
	app := getSignalTestConfig(t)
	bus := app.Bus
	app.Jobs[0].Run(bus)

	ds := mocks.NewDebugSubscriber(bus, 4)
	ds.Run(0)

	app.Terminate()
	ds.Close()

	got := map[events.Event]int{}
	for _, result := range ds.Results {
		got[result]++
	}
	if !reflect.DeepEqual(got, map[events.Event]int{
		events.GlobalShutdown:                                       1,
		events.QuitByClose:                                          1,
		events.Event{Code: events.Stopping, Source: "test-service"}: 1,
		events.Event{Code: events.Stopped, Source: "test-service"}:  1,
	}) {
		t.Fatalf("expected shutdown but got:\n%v", ds.Results)
	}
}

// Test handler for SIGHUP // TODO this only tests the reload method
func TestReloadSignal(t *testing.T) {
	app := getSignalTestConfig(t)

	// write invalid config to temp file and assign it as app config
	f := testCfgToTempFile(t, `invalid`)
	defer os.Remove(f.Name())
	app.ConfigFlag = f.Name()

	err := app.reload()
	if err == nil {
		t.Errorf("invalid configuration did not return error")
	}

	// write new valid configuration
	validConfig := []byte(`{ "consul": "newconsul:8500" }`)
	f2, err := os.Create(f.Name()) // we'll just blow away the old file
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f2.Write(validConfig); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}
	err = app.reload()
	if err != nil {
		t.Errorf("valid configuration returned error: %v", err)
	}
	discSvc := app.Discovery
	if svc, ok := discSvc.(*consul.Consul); !ok || svc == nil {
		t.Errorf("configuration was not reloaded: %v", discSvc)
	}
}

// Test that only ensures that we cover a straight-line run through
// the handleSignals setup code
func TestSignalWiring(t *testing.T) {
	app := EmptyApp()
	app.Bus = events.NewEventBus()
	app.handleSignals()
	sendAndWaitForSignal(t, syscall.SIGUSR1)
	sendAndWaitForSignal(t, syscall.SIGTERM)
	sendAndWaitForSignal(t, syscall.SIGCHLD)
	sendAndWaitForSignal(t, syscall.SIGHUP)
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
