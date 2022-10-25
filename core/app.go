// Package core contains the main control loop.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tritondatacenter/containerpilot/config"
	"github.com/tritondatacenter/containerpilot/control"
	"github.com/tritondatacenter/containerpilot/discovery"
	"github.com/tritondatacenter/containerpilot/events"
	"github.com/tritondatacenter/containerpilot/jobs"
	"github.com/tritondatacenter/containerpilot/telemetry"
	"github.com/tritondatacenter/containerpilot/watches"

	log "github.com/sirupsen/logrus"
)

// App encapsulates the state of ContainerPilot after the initial setup.
type App struct {
	ControlServer *control.HTTPServer
	Discovery     discovery.Backend
	Jobs          []*jobs.Job
	Watches       []*watches.Watch
	Telemetry     *telemetry.Telemetry
	StopTimeout   int
	signalLock    *sync.RWMutex
	ConfigFlag    string
	Bus           *events.EventBus
}

// EmptyApp creates an empty application
func EmptyApp() *App {
	app := &App{}
	app.signalLock = &sync.RWMutex{}
	return app
}

// NewApp creates a new App from the config
func NewApp(configFlag string) (*App, error) {
	os.Setenv("CONTAINERPILOT_PID", fmt.Sprintf("%v", os.Getpid()))
	a := EmptyApp()
	cfg, err := config.LoadConfig(configFlag)
	if err != nil {
		return nil, err
	}

	if err := cfg.InitLogging(); err != nil {
		return nil, err
	}
	if log.GetLevel() >= log.DebugLevel {
		configJSON, err := json.Marshal(cfg)
		if err != nil {
			log.Errorf("error marshalling config for debug: %v", err)
		}
		log.Debugf("loaded config: %v", string(configJSON))
	}

	cs, err := control.NewHTTPServer(cfg.Control)
	if err != nil {
		return nil, err
	}
	a.ControlServer = cs

	a.StopTimeout = cfg.StopTimeout
	a.Discovery = cfg.Discovery
	a.Jobs = jobs.FromConfigs(cfg.Jobs)
	a.Watches = watches.FromConfigs(cfg.Watches)
	a.Telemetry = telemetry.NewTelemetry(cfg.Telemetry)
	a.Telemetry.MonitorJobs(a.Jobs)
	a.Telemetry.MonitorWatches(a.Watches)
	a.ConfigFlag = configFlag // stash the old config

	// set an environment variable for each job IP address so that
	// forked processes have access to this information
	for _, job := range a.Jobs {
		if job.Service != nil {
			envKey := getEnvVarNameFromService(job.Name)
			os.Setenv(envKey, job.Service.IPAddress)
		}
	}

	return a, nil
}

// Normalize the validated service name as an environment variable
func getEnvVarNameFromService(service string) string {
	envKey := strings.ToUpper(service)
	envKey = strings.Replace(envKey, "-", "_", -1)
	envKey = fmt.Sprintf("CONTAINERPILOT_%v_IP", envKey)
	return envKey
}

// Run starts the application and blocks until finished
func (a *App) Run() {
	a.handleSignals()

	for {
		ctx, cancel := context.WithCancel(context.Background())

		// This provides an escape hatch for shutting down after all jobs have
		// completed. Each time a job completes, during its cleanup func, it
		// will be set `IsComplete` to `true`. Then it'll send `true` through
		// this channel to wake up this goroutine, check all jobs for
		// `IsComplete`, and shutdown ContainerPilot if all jobs are
		// completed. This is because ContainerPilot is NOT a server and must
		// shut down when no longer required (i.e. process containers).
		//
		// Consider that this fires the global context.CancelFunc after all jobs
		// have completed. This means we'll serialize the shut down of all
		// dependent services, control server, telemetry, watches, metrics,
		// after jobs have finalized here (quit == true). This is the reason the
		// signal handler above, and reload endpoint, only need to fire a
		// GlobalShutdown across the event bus. Context handles everything after
		// that process.
		completedCh := make(chan struct{}, 1)
		go func() {
			for {
				select {
				case <-completedCh:
					quit := true
					for _, job := range a.Jobs {
						if !job.IsComplete {
							quit = false
						}
					}
					if quit {
						cancel()
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()

		a.Bus = events.NewEventBus()
		a.ControlServer.Run(ctx, a.Bus)
		a.runTasks(ctx, completedCh)

		if !a.Bus.Wait() {
			if a.StopTimeout > 0 {
				log.Debugf("killing all processes in %v seconds", a.StopTimeout)
				tick := time.NewTimer(time.Duration(a.StopTimeout) * time.Second)
				<-tick.C
			}
			for _, job := range a.Jobs {
				log.Infof("killing processes for job %#v", job.Name)
				job.Kill()
			}
			break
		}
		if err := a.reload(); err != nil {
			log.Error(err)
			break
		}
		close(completedCh)
	}
}

// Terminate kills the application
func (a *App) Terminate() {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	a.Bus.Shutdown()
}

// SignalEvent publishes a signal event onto the event bus
func (a *App) SignalEvent(sig string) {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()
	a.Bus.PublishSignal(sig)
}

// reload does the actual work of reloading the configuration and
// updating the App with those changes. The EventBus should be
// already shut down before we call this.
func (a *App) reload() error {
	newApp, err := NewApp(a.ConfigFlag)
	if err != nil {
		log.Errorf("error initializing config: %v", err)
		return err
	}
	a.Discovery = newApp.Discovery
	a.Jobs = newApp.Jobs
	a.Watches = newApp.Watches
	a.StopTimeout = newApp.StopTimeout
	a.Telemetry = newApp.Telemetry
	a.ControlServer = newApp.ControlServer
	return nil
}

// HandlePolling sets up polling functions and write their quit channels
// back to our config
func (a *App) runTasks(ctx context.Context, completedCh chan struct{}) {
	// we need to subscribe to events before we Run all the jobs
	// to avoid races where a job finishes and fires events before
	// other jobs are even subscribed to listen for them.
	for _, job := range a.Jobs {
		job.Subscribe(a.Bus)
		job.Register(a.Bus)
	}
	for _, job := range a.Jobs {
		job.Run(ctx, completedCh)
	}
	for _, watch := range a.Watches {
		watch.Run(ctx, a.Bus)
	}
	if a.Telemetry != nil {
		for _, metric := range a.Telemetry.Metrics {
			metric.Run(ctx, a.Bus)
		}
		a.Telemetry.Run(ctx)
	}
	// kick everything off
	a.Bus.Publish(events.GlobalStartup)
}
