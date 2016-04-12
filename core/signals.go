package core

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerbuddy/config"
	"github.com/joyent/containerbuddy/services"
	"github.com/joyent/containerbuddy/utils"
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

func toggleMaintenanceMode(cfg *config.Config) {
	maintModeLock.RLock()
	signalLock.Lock()
	defer signalLock.Unlock()
	defer maintModeLock.RUnlock()
	paused = !paused
	if paused {
		forAllServices(cfg, func(service *services.Service) {
			log.Infof("Marking for maintenance: %s", service.Name)
			service.MarkForMaintenance()
		})
	}
}

func terminate(cfg *config.Config) {
	signalLock.Lock()
	defer signalLock.Unlock()
	stopPolling(cfg)
	forAllServices(cfg, func(service *services.Service) {
		log.Infof("Deregistering service: %s", service.Name)
		service.Deregister()
	})

	// Run and wait for preStop command to exit
	utils.Run(cfg.PreStopCmd)

	cmd := cfg.Command
	if cmd == nil || cmd.Process == nil {
		// Not managing the process, so don't do anything
		return
	}
	if cfg.StopTimeout > 0 {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Warnf("Error sending SIGTERM to application: %s", err)
		} else {
			time.AfterFunc(time.Duration(cfg.StopTimeout)*time.Second, func() {
				log.Infof("Killing Process %#v", cmd.Process)
				cmd.Process.Kill()
			})
			return
		}
	}
	log.Infof("Killing Process %#v", cfg.Command.Process)
	cmd.Process.Kill()
}

func reloadConfig(cfg *config.Config) *config.Config {
	signalLock.Lock()
	defer signalLock.Unlock()
	log.Infof("Reloading configuration.")
	newConfig, err := config.ReloadConfig(cfg.ConfigFlag)
	if err != nil {
		log.Errorf("Could not reload config: %v", err)
		return nil
	}
	// stop advertising the existing services so that we can
	// make sure we update them if ports, etc. change.
	stopPolling(cfg)
	forAllServices(cfg, func(service *services.Service) {
		log.Infof("Deregistering service: %s", service.Name)
		service.Deregister()
	})

	signal.Reset()
	HandleSignals(newConfig)
	HandlePolling(newConfig)

	return newConfig // return for debuggability
}

func stopPolling(cfg *config.Config) {
	for _, quit := range cfg.QuitChannels {
		quit <- true
	}
}

type serviceFunc func(service *services.Service)

func forAllServices(cfg *config.Config, fn serviceFunc) {
	for _, service := range cfg.Services {
		fn(service)
	}
}

// Listen for and capture signals used for orchestration
func HandleSignals(cfg *config.Config) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for signal := range sig {
			switch signal {
			case syscall.SIGUSR1:
				toggleMaintenanceMode(cfg)
			case syscall.SIGTERM:
				terminate(cfg)
			case syscall.SIGHUP:
				reloadConfig(cfg)
			}
		}
	}()
}

// on SIGCHLD send wait4() (ref http://linux.die.net/man/2/waitpid)
// to clean up any potential zombies
func ReapChildren() {
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
