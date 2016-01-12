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
var signalLock = sync.RWMutex{}
var maintModeLock = sync.RWMutex{}

// we wrap access to `paused` in a RLock so that if we're in the middle of
// marking services for maintenance we don't get stale reads
func inMaintenanceMode() bool {
	maintModeLock.RLock()
	defer maintModeLock.RUnlock()
	return paused
}

func toggleMaintenanceMode(config *Config) {
	maintModeLock.RLock()
	signalLock.Lock()
	defer signalLock.Unlock()
	defer maintModeLock.RUnlock()
	paused = !paused
	if paused {
		forAllServices(config, func(service *ServiceConfig) {
			log.Printf("Marking for maintenance: %s\n", service.Name)
			service.MarkForMaintenance()
		})
	}
}

func terminate(config *Config) {
	signalLock.Lock()
	defer signalLock.Unlock()
	stopPolling(config)
	forAllServices(config, func(service *ServiceConfig) {
		log.Printf("Deregistering service: %s\n", service.Name)
		service.Deregister()
	})

	// Run and wait for preStop command to exit
	run(config.preStopCmd)

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

func reloadConfig(config *Config) *Config {
	signalLock.Lock()
	defer signalLock.Unlock()
	log.Printf("Reloading configuration.\n")
	newConfig, err := loadConfig()
	if err != nil {
		log.Printf("Could not reload config: %v\n", err)
		return nil
	}
	// stop advertising the existing services so that we can
	// make sure we update them if ports, etc. change.
	stopPolling(config)
	forAllServices(config, func(service *ServiceConfig) {
		log.Printf("Deregistering service: %s\n", service.Name)
		service.Deregister()
	})

	signal.Reset()
	handleSignals(newConfig)
	handlePolling(newConfig)

	return newConfig // return for debuggability
}

func stopPolling(config *Config) {
	for _, quit := range config.QuitChannels {
		quit <- true
	}
}

type serviceFunc func(service *ServiceConfig)

func forAllServices(config *Config, fn serviceFunc) {
	for _, service := range config.Services {
		fn(service)
	}
}

// Listen for and capture signals from the OS
func handleSignals(config *Config) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for signal := range sig {
			switch signal {
			case syscall.SIGUSR1:
				toggleMaintenanceMode(config)
			case syscall.SIGTERM:
				terminate(config)
			case syscall.SIGHUP:
				reloadConfig(config)
			}
		}
	}()
}
