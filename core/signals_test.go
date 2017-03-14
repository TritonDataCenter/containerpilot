package core

import (
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/discovery/consul"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/services"
)

// ------------------------------------------
// Test setup with mock services

// Mock Discovery
type NoopServiceBackend struct{}

func (c *NoopServiceBackend) SendHeartbeat(service *discovery.ServiceDefinition)      { return }
func (c *NoopServiceBackend) CheckForUpstreamChanges(backend, tag string) bool        { return false }
func (c *NoopServiceBackend) MarkForMaintenance(service *discovery.ServiceDefinition) {}
func (c *NoopServiceBackend) Deregister(service *discovery.ServiceDefinition)         {}

func getSignalTestConfig(t *testing.T) *App {

	cfg := &services.ServiceConfig{
		ID:         "test-service",
		Name:       "test-service",
		Heartbeat:  1,
		Port:       1,
		TTL:        1,
		Interfaces: []string{"inet"},
		Exec:       []string{"./testdata/test.sh", "interruptSleep"},
	}
	cfg.Validate(&NoopServiceBackend{})
	service := services.NewService(cfg)
	app := EmptyApp()
	app.StopTimeout = 5
	app.Services = []*services.Service{service}
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
	app.Services[0].Run(bus)

	ds := events.NewDebugSubscriber(bus, 4)
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

// Test handler for SIGHUP
func TestReloadSignal(t *testing.T) {
	app := getSignalTestConfig(t)
	app.ConfigFlag = "invalid"
	err := app.Reload()
	if err == nil {
		t.Errorf("Invalid configuration did not return error")
	}
	app.ConfigFlag = `{ "consul": "newconsul:8500" }`
	err = app.Reload()
	if err != nil {
		t.Errorf("Valid configuration returned error: %v", err)
	}
	discSvc := app.Discovery
	if svc, ok := discSvc.(*consul.Consul); !ok || svc == nil {
		t.Errorf("Configuration was not reloaded: %v", discSvc)
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
