package core

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/discovery/consul"
	"github.com/joyent/containerpilot/services"
	"github.com/joyent/containerpilot/utils"
)

// ------------------------------------------
// Test setup with mock services

// Mock Discovery
type NoopDiscoveryService struct{}

func (c *NoopDiscoveryService) SendHeartbeat(service *discovery.ServiceDefinition)      { return }
func (c *NoopDiscoveryService) CheckForUpstreamChanges(backend, tag string) bool        { return false }
func (c *NoopDiscoveryService) MarkForMaintenance(service *discovery.ServiceDefinition) {}
func (c *NoopDiscoveryService) Deregister(service *discovery.ServiceDefinition)         {}

func getSignalTestConfig() *App {
	service, _ := services.NewService(
		"test-service", 1, 1, 1, nil, nil, &NoopDiscoveryService{})
	app := EmptyApp()
	app.Command = utils.ArgsToCmd([]string{
		"./testdata/test.sh",
		"interruptSleep"})
	app.StopTimeout = 5
	app.Services = []*services.Service{service}
	return app
}

// ------------------------------------------

// Test handler for SIGUSR1
func TestMaintenanceSignal(t *testing.T) {
	app := getSignalTestConfig()

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

	app := getSignalTestConfig()
	startTime := time.Now()
	go func() {
		if exitCode, _ := utils.ExecuteAndWait(app.Command); exitCode != 2 {
			t.Fatalf("Expected exit code 2 but got %d", exitCode)
		}
	}()
	// we need time for the forked process to start up and this is async
	runtime.Gosched()
	time.Sleep(10 * time.Millisecond)

	app.Terminate()
	elapsed := time.Since(startTime)
	if elapsed.Seconds() > float64(app.StopTimeout) {
		t.Fatalf("Expected elapsed time <= %d seconds, but was %.2f",
			app.StopTimeout, elapsed.Seconds())
	}
}

// Test handler for SIGHUP
func TestReloadSignal(t *testing.T) {
	app := getSignalTestConfig()
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
	discSvc := app.DiscoveryService
	if svc, ok := discSvc.(*consul.Consul); !ok || svc == nil {
		t.Errorf("Configuration was not reloaded: %v", discSvc)
	}
}

// Test that only ensures that we cover a straight-line run through
// the handleSignals setup code
func TestSignalWiring(t *testing.T) {
	app := EmptyApp()
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
