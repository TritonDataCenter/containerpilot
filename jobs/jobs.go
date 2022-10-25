// Package jobs manages the configuration and execution of the jobs
package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tritondatacenter/containerpilot/commands"
	"github.com/tritondatacenter/containerpilot/discovery"
	"github.com/tritondatacenter/containerpilot/events"
	log "github.com/sirupsen/logrus"
)

type processEventStatus bool

// Some magic numbers used internally by processEvent
const (
	unlimited                          = -1
	jobContinue     processEventStatus = false
	jobHalt         processEventStatus = true
	eventBufferSize                    = 1000
)

// Job manages the state of a job and its start/stop conditions
type Job struct {
	Name string
	exec *commands.Command

	// service health and discovery
	Status          JobStatus
	statusLock      *sync.RWMutex
	Service         *discovery.ServiceDefinition
	healthCheckExec *commands.Command
	// staticcheck U1000 field is unused
	//healthCheckName string

	// starting events
	startEvent        events.Event
	startTimeout      time.Duration
	startsRemain      int
	startTimeoutEvent events.Event

	// stopping events
	stoppingWaitEvent events.Event
	stoppingTimeout   time.Duration

	// timing and restarts
	heartbeat      time.Duration
	restartLimit   int
	restartsRemain int
	frequency      time.Duration

	// completed
	IsComplete   bool
	completeLock *sync.RWMutex

	events.Subscriber
	events.Publisher
}

// NewJob creates a new Job from a Config
func NewJob(cfg *Config) *Job {
	job := &Job{
		Name:              cfg.Name,
		exec:              cfg.exec,
		heartbeat:         cfg.heartbeatInterval,
		Service:           cfg.serviceDefinition,
		healthCheckExec:   cfg.healthCheckExec,
		startEvent:        cfg.whenEvent,
		startTimeout:      cfg.whenTimeout,
		startsRemain:      cfg.whenStartsLimit,
		stoppingWaitEvent: cfg.stoppingWaitEvent,
		stoppingTimeout:   cfg.stoppingTimeout,
		restartLimit:      cfg.restartLimit,
		restartsRemain:    cfg.restartLimit,
		frequency:         cfg.freqInterval,
	}
	job.statusLock = &sync.RWMutex{}
	job.completeLock = &sync.RWMutex{}
	job.Rx = make(chan events.Event, eventBufferSize)
	if job.Name == "containerpilot" {
		// right now this hardcodes the telemetry service to
		// be always "healthy", but maybe we want to have it verify itself
		// before heartbeating in the future?
		job.setStatus(statusAlwaysHealthy)
	}
	return job
}

// FromConfigs creates Jobs from a slice of validated Configs
func FromConfigs(cfgs []*Config) []*Job {
	jobs := []*Job{}
	for _, cfg := range cfgs {
		job := NewJob(cfg)
		jobs = append(jobs, job)
	}
	return jobs
}

// SendHeartbeat sends a heartbeat for this Job's service
func (job *Job) SendHeartbeat() {
	if job.Service != nil {
		job.Service.SendHeartbeat()
	}
}

// checkRegistration registers this Job's service if it isn't already registered.
func (job *Job) checkRegistration() {
	if job.Service != nil && job.Service.InitialStatus != "" {
		job.Service.RegisterWithInitialStatus()
	}
}

// GetStatus returns the current health status of the Job
func (job *Job) GetStatus() JobStatus {
	job.statusLock.RLock()
	defer job.statusLock.RUnlock()
	return job.Status
}

func (job *Job) setStatus(status JobStatus) {
	job.statusLock.Lock()
	defer job.statusLock.Unlock()
	if job.Status != statusAlwaysHealthy {
		job.Status = status
	}
}

func (job *Job) setComplete() {
	job.completeLock.Lock()
	defer job.completeLock.Unlock()
	job.IsComplete = true
}

// Kill sends SIGTERM to the Job's executable, if any
func (job *Job) Kill() {
	if job.exec != nil {
		job.exec.Kill()
	}
}

// Run executes the event loop for the Job
func (job *Job) Run(pctx context.Context, completedCh chan struct{}) {
	ctx, cancel := context.WithCancel(pctx)

	if job.frequency > 0 {
		events.NewEventTimer(ctx, job.Rx, job.frequency,
			fmt.Sprintf("%s.run-every", job.Name))
	}
	if job.heartbeat > 0 {
		events.NewEventTimer(ctx, job.Rx, job.heartbeat,
			fmt.Sprintf("%s.heartbeat", job.Name))
	}
	if job.startTimeout > 0 {
		timeoutName := fmt.Sprintf("%s.wait-timeout", job.Name)
		events.NewEventTimeout(ctx, job.Rx, job.startTimeout, timeoutName)
		job.startTimeoutEvent = events.Event{Code: events.TimerExpired, Source: timeoutName}
	} else {
		job.startTimeoutEvent = events.NonEvent
	}

	go func() {
		defer func() {
			job.cleanup(ctx, cancel)
			completedCh <- struct{}{}
		}()
		for {
			// Check if job's service has been registered. Doing it inside the event
			// loop to retry if consul registration fails.
			job.checkRegistration()
			select {
			case event, ok := <-job.Rx:
				if !ok || event == events.QuitByTest {
					return
				}
				if job.processEvent(ctx, event) == jobHalt {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (job *Job) processEvent(ctx context.Context, event events.Event) processEventStatus {
	runEverySource := fmt.Sprintf("%s.run-every", job.Name)
	heartbeatSource := fmt.Sprintf("%s.heartbeat", job.Name)
	healthCheckName := fmt.Sprintf("check.%s", job.Name)
	if job.healthCheckExec != nil {
		healthCheckName = job.healthCheckExec.Name
	}

	switch event {

	case events.Event{Code: events.TimerExpired, Source: heartbeatSource}:
		return job.onHeartbeatTimerExpired(ctx)

	case job.startTimeoutEvent:
		return job.onStartTimeoutExpired(ctx)

	case events.Event{Code: events.TimerExpired, Source: runEverySource}:
		return job.onRunEveryTimerExpired(ctx)

	case events.Event{Code: events.ExitFailed, Source: healthCheckName}:
		return job.onHealthCheckFailed(ctx)

	case events.Event{Code: events.ExitSuccess, Source: healthCheckName}:
		return job.onHealthCheckPassed(ctx)

	case events.Event{Code: events.Quit, Source: job.Name},
		events.GlobalShutdown:
		return job.onQuit(ctx)

	case events.GlobalEnterMaintenance:
		return job.onEnterMaintenance(ctx)

	case events.GlobalExitMaintenance:
		return job.onExitMaintenance(ctx)

	case events.Event{Code: events.ExitSuccess, Source: job.Name},
		events.Event{Code: events.ExitFailed, Source: job.Name}:
		return job.onExecExit(ctx)

	case events.Event{Code: events.Signal, Source: "SIGHUP"},
		events.Event{Code: events.Signal, Source: "SIGUSR2"}:
		return job.onSignalEvent(ctx, event.Source)

	case job.startEvent:
		return job.onStartEvent(ctx)
	}
	return jobContinue
}

// startJobExec runs the Job's executable and returns without waiting
func (job *Job) startJobExec(ctx context.Context) {
	job.startTimeoutEvent = events.NonEvent
	job.setStatus(statusUnknown)
	if job.exec != nil {
		job.exec.Run(ctx, job.Publisher.Bus)
	}
}

func (job *Job) onHeartbeatTimerExpired(ctx context.Context) processEventStatus {
	status := job.GetStatus()
	if status != statusMaintenance && status != statusIdle {
		if job.healthCheckExec != nil {
			job.healthCheckExec.Run(ctx, job.Publisher.Bus)
		} else if job.Service != nil {
			// this is the case for non-checked but advertised
			// services like the telemetry endpoint
			job.SendHeartbeat()
		}
	}
	return jobContinue
}

func (job *Job) onStartTimeoutExpired(ctx context.Context) processEventStatus {
	job.Publish(events.Event{
		Code: events.TimerExpired, Source: job.Name})
	job.Rx <- events.Event{Code: events.Quit, Source: job.Name}
	return jobContinue
}

func (job *Job) onRunEveryTimerExpired(ctx context.Context) processEventStatus {
	if !job.restartPermitted() {
		log.Debugf("interval expired but restart not permitted: %v",
			job.Name)
		job.startEvent = events.NonEvent
		return jobHalt
	}
	job.restartsRemain--
	job.startJobExec(ctx)
	return jobContinue
}

func (job *Job) onHealthCheckFailed(ctx context.Context) processEventStatus {
	if job.GetStatus() != statusMaintenance {
		job.setStatus(statusUnhealthy)
		job.Publish(events.Event{Code: events.StatusUnhealthy, Source: job.Name})
	}
	return jobContinue
}

func (job *Job) onHealthCheckPassed(ctx context.Context) processEventStatus {
	if job.GetStatus() != statusMaintenance {
		job.setStatus(statusHealthy)
		job.Publish(events.Event{Code: events.StatusHealthy, Source: job.Name})
		job.SendHeartbeat()
	}
	return jobContinue
}

func (job *Job) onQuit(ctx context.Context) processEventStatus {
	job.restartsRemain = 0 // no more restarts
	if (job.startEvent.Code == events.Stopping ||
		job.startEvent.Code == events.Stopped) &&
		job.exec != nil {
		// "pre-stop" and "post-stop" style jobs ignore the global
		// shutdown and return on their ExitSuccess/ExitFailed.
		// if the stop timeout on the global shutdown is exceeded
		// the whole process gets SIGKILL
		if job.startsRemain == unlimited {
			job.startsRemain = 1
		}
		return jobContinue
	}
	job.startsRemain = 0
	job.startEvent = events.NonEvent
	return jobHalt
}

func (job *Job) onEnterMaintenance(ctx context.Context) processEventStatus {
	job.setStatus(statusMaintenance)
	if job.Service != nil {
		job.Service.MarkForMaintenance()
	}
	if job.startEvent == events.GlobalEnterMaintenance {
		return job.onStartEvent(ctx)
	}
	return jobContinue
}

func (job *Job) onExitMaintenance(ctx context.Context) processEventStatus {
	job.setStatus(statusUnknown)
	if job.startEvent == events.GlobalExitMaintenance {
		return job.onStartEvent(ctx)
	}
	return jobContinue
}

func (job *Job) onExecExit(ctx context.Context) processEventStatus {
	if job.frequency > 0 {
		return jobContinue // periodic jobs ignore previous events
	}
	if job.restartPermitted() {
		job.restartsRemain--
		job.startJobExec(ctx)
		return jobContinue
	}
	if job.startsRemain != 0 {
		return jobContinue
	}
	log.Debugf("job exited but restart not permitted: %v", job.Name)
	job.startEvent = events.NonEvent
	job.setStatus(statusUnknown)
	return jobHalt
}

func (job *Job) onSignalEvent(ctx context.Context, sig string) processEventStatus {
	if job.startEvent.Code == events.Signal &&
		job.startEvent.Source == sig {
		job.startJobExec(ctx)
	}
	return jobContinue
}

func (job *Job) onStartEvent(ctx context.Context) processEventStatus {
	if job.startsRemain == 0 {
		job.startEvent = events.NonEvent
		return jobHalt
	}
	if job.startsRemain != unlimited {
		// if we have unlimited restarts we want to make sure we don't
		// decrement forever and then wrap-around
		job.startsRemain--
		if job.startsRemain == 0 || job.restartsRemain == 0 {
			// prevent ourselves from receiving the start event again
			// if it fires while we're still running the job's exec
			job.startEvent = events.NonEvent
		}
	}
	job.startJobExec(ctx)
	return jobContinue
}

func (job *Job) restartPermitted() bool {
	if job.restartLimit == unlimited || job.restartsRemain > 0 {
		return true
	}
	return false
}

// cleanup fires the Stopping event and will wait to receive a stoppingWaitEvent
// if one is configured. cleans up registration to event bus and closes all
// channels and contexts when done.
func (job *Job) cleanup(ctx context.Context, cancel context.CancelFunc) {
	stoppingTimeout := fmt.Sprintf("%s.stopping-timeout", job.Name)
	job.Publish(events.Event{Code: events.Stopping, Source: job.Name})
	if job.stoppingWaitEvent != events.NonEvent {
		if job.stoppingTimeout > 0 {
			// not having this set is a programmer error not a runtime error
			events.NewEventTimeout(ctx, job.Rx,
				job.stoppingTimeout, stoppingTimeout)
		}
	loop:
		for {
			event := <-job.Rx
			switch event {
			case job.stoppingWaitEvent:
				break loop
			case events.Event{Code: events.Stopping, Source: stoppingTimeout}:
				break loop
			}
		}
	}
	cancel()
	if job.Service != nil {
		job.Service.Deregister() // deregister from Consul
	}
	job.Unsubscribe() // deregister from events
	job.Unregister()
	job.setComplete()
	job.Publish(events.Event{Code: events.Stopped, Source: job.Name})
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (job *Job) String() string {
	return "jobs.Job[" + job.Name + "]"
}
