package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"
)

func TestMaintenanceSignal(t *testing.T) {

	if inMaintenanceMode() {
		t.Errorf("Should not be in maintenance mode before starting handler")
	}
	handleSignals(&Config{})
	if inMaintenanceMode() {
		t.Errorf("Should not be in maintenance mode after starting handler")
	}

	sendAndWaitForSignal(t, syscall.SIGUSR1)
	if !inMaintenanceMode() {
		t.Errorf("Should be in maintenance mode after receiving SIGUSR1")
	}
	sendAndWaitForSignal(t, syscall.SIGUSR1)
	if inMaintenanceMode() {
		t.Errorf("Should not be in maintenance mode after receiving second SIGUSR1")
	}
}

// helper to ensure that we block for the signal to deliver any signal
// we need, and then yield execution so that the handler gets a chance
// at running. If we don't do this there's a race where we can check
// resulting side-effects of a handler before it's been run
func sendAndWaitForSignal(t *testing.T, s os.Signal) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	me, _ := os.FindProcess(os.Getpid())
	if err := me.Signal(s); err != nil {
		t.Errorf("Got error on SIGUSR1: %v", err)
	}
	<-sig
	runtime.Gosched()
}
