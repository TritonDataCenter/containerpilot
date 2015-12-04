package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
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

func terminate(config *Config) {
	cmd := config.Command
	if cmd == nil || cmd.Process == nil {
		// Not managing the process, so don't do anything
		return
	}
	if config.StopTimeout > 0 {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("Error sending SIGTERM to application: %s\n", err)
		} else {
			time.AfterFunc(time.Duration(config.StopTimeout)*time.Second, func() {
				log.Printf("Killing Process %#v\n", cmd.Process)
				cmd.Process.Kill()
			})
			return
		}
	}
	log.Printf("Killing Process %#v\n", config.Command.Process)
	cmd.Process.Kill()
}

func stopPolling(config *Config) {
	for _, quit := range config.QuitChannels {
		quit <- true
	}
}

func deregisterServices(config *Config) {
	log.Println("Deregister All Services")
	for _, service := range config.Services {
		log.Printf("Deregister service: %s\n", service.Name)
		service.Deregister()
	}
}

// Listen for and capture signals from the OS
func handleSignals(config *Config) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGTERM)
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
			case syscall.SIGTERM:
				log.Println("Caught SIGTERM")
				stopPolling(config)
				deregisterServices(config)
				terminate(config)
			}
		}
	}()
}
