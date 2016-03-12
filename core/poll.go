package core

import (
	"time"

	"github.com/joyent/containerpilot/config"
)

// Set up polling functions and write their quit channels
// back to our config
func HandlePolling(cfg *config.Config) {
	var quit []chan bool
	for _, backend := range cfg.Backends {
		quit = append(quit, poll(backend))
	}
	for _, service := range cfg.Services {
		quit = append(quit, poll(service))
	}
	if cfg.Telemetry != nil {
		for _, sensor := range cfg.Telemetry.Sensors {
			quit = append(quit, poll(sensor))
		}
		cfg.Telemetry.Serve()
	}
	if cfg.Tasks != nil {
		for _, task := range cfg.Tasks {
			task.Start()
		}
	}
	cfg.QuitChannels = quit
}

// Every `pollTime` seconds, run the `PollingFunc` function.
// Expect a bool on the quit channel to stop gracefully.
func poll(pollable Pollable) chan bool {
	ticker := time.NewTicker(time.Duration(pollable.PollTime()) * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !inMaintenanceMode() {
					pollable.PollAction()
				}
			case <-quit:
				return
			}
		}
	}()
	return quit
}

// Pollable is base abstraction for backends and services that support polling
type Pollable interface {
	PollTime() int
	PollAction()
}
