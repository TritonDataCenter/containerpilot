package containerbuddy

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
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
			log.Infof("Marking for maintenance: %s", service.Name)
			service.MarkForMaintenance()
		})
	}
}

func terminate(config *Config) {
	signalLock.Lock()
	defer signalLock.Unlock()
	stopPolling(config)
	forAllServices(config, func(service *ServiceConfig) {
		log.Infof("Deregistering service: %s", service.Name)
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
			log.Warnf("Error sending SIGTERM to application: %s", err)
		} else {
			time.AfterFunc(time.Duration(config.StopTimeout)*time.Second, func() {
				log.Infof("Killing Process %#v", cmd.Process)
				cmd.Process.Kill()
			})
			return
		}
	}
	log.Infof("Killing Process %#v", config.Command.Process)
	cmd.Process.Kill()
}

func reloadConfig(config *Config) *Config {
	signalLock.Lock()
	defer signalLock.Unlock()
	log.Infof("Reloading configuration.")
	newConfig, err := loadConfig()
	if err != nil {
		log.Errorf("Could not reload config: %v", err)
		return nil
	}
	// stop advertising the existing services so that we can
	// make sure we update them if ports, etc. change.
	stopPolling(config)
	forAllServices(config, func(service *ServiceConfig) {
		log.Infof("Deregistering service: %s", service.Name)
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

// Listen for and capture signals used for orchestration
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

// on SIGCHLD send wait4() (ref http://linux.die.net/man/2/waitpid)
// to clean up any potential zombies
func reapChildren() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGCHLD)
	go func() {
		// wait for signals on the channel until it closes
		for _ = range sig {
			for {
				// only 1 SIGCHLD can be handled at a time from the channel,
				// so we need to allow for the possibility that multiple child
				// processes have terminated while one is already being reaped.
				var wstatus syscall.WaitStatus
				if _, err := syscall.Wait4(-1, &wstatus,
					syscall.WNOHANG|syscall.WUNTRACED|syscall.WCONTINUED,
					nil); err == syscall.EINTR {
					continue
				}
				// return to the outer loop and wait for another signal
				break
			}
		}
	}()
}
