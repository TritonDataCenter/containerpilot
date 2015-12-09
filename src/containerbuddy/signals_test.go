package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// ------------------------------------------
// Test setup with mock services

// Mock Discovery
type NoopDiscoveryService struct{}

func (c *NoopDiscoveryService) SendHeartbeat(service *ServiceConfig) { return }
func (c *NoopDiscoveryService) CheckForUpstreamChanges(backend *BackendConfig) bool {
	return false
}

func (c *NoopDiscoveryService) MarkForMaintenance(service *ServiceConfig) {}
func (c *NoopDiscoveryService) Deregister(service *ServiceConfig)         {}

func getSignalTestConfig() *Config {
	config := &Config{
		Command: argsToCmd([]string{
			"/root/examples/test/test.sh",
			"interruptSleep"}),
		StopTimeout: 5,
		Services: []*ServiceConfig{
			&ServiceConfig{
				Name:             "test-service",
				Poll:             1,
				discoveryService: &NoopDiscoveryService{},
			},
		},
	}
	return config
}

// ------------------------------------------

// Test handler for SIGUSR1
func TestMaintenanceSignal(t *testing.T) {
	config := getSignalTestConfig()
	if inMaintenanceMode() {
		t.Fatal("Should not be in maintenance mode by default")
	}

	toggleMaintenanceMode(config)
	if !inMaintenanceMode() {
		t.Fatal("Should be in maintenance mode after receiving SIGUSR1")
	}

	toggleMaintenanceMode(config)
	if inMaintenanceMode() {
		t.Fatal("Should not be in maintenance mode after receiving second SIGUSR1")
	}
}

// Test handler for SIGTERM
func TestTerminateSignal(t *testing.T) {

	config := getSignalTestConfig()
	startTime := time.Now()
	quit := make(chan int, 1)
	go func() {
		if exitCode, _ := executeAndWait(config.Command); exitCode != 2 {
			t.Fatalf("Expected exit code 2 but got %d", exitCode)
		}
		quit <- 1
	}()
	// we need time for the forked process to start up and this is async
	runtime.Gosched()
	time.Sleep(10 * time.Millisecond)

	terminate(config)
	close(quit)
	elapsed := time.Since(startTime)
	if elapsed.Seconds() > float64(config.StopTimeout) {
		t.Fatalf("Expected elapsed time <= %d seconds, but was %.2f",
			config.StopTimeout, elapsed.Seconds())
	}
}

// Test that only ensures that we cover a straight-line run through
// the handleSignals setup code
func TestSignalWiring(t *testing.T) {
	handleSignals(&Config{})
	sendAndWaitForSignal(t, syscall.SIGUSR1)
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
