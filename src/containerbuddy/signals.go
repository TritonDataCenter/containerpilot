package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// globals are eeeeevil
var paused bool
var pauseLock = sync.RWMutex{}

// we wrap access to `paused` in a RLock so that if we're in the middle of
// marking services for maintenance we don't get stale reads
func inMaintenanceMode() bool {
	pauseLock.RLock()
	defer pauseLock.RUnlock()
	return paused
}

func toggleMaintenanceMode() {
	pauseLock.Lock()
	defer pauseLock.Unlock()
	paused = !paused
}

// Listen for and capture signals from the OS
func handleSignals(config *Config) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for signal := range sig {
			switch signal {
			// there's only one handler today but this makes it obvious
			// where to add support for new signals
			case syscall.SIGUSR1:
				toggleMaintenanceMode()
				if inMaintenanceMode() {
					log.Println("we are paused!")
					for _, service := range config.Services {
						log.Printf("Marking for maintenance: %s\n", service.Name)
						service.MarkForMaintenance()
					}
				}
			}
		}
	}()
}
