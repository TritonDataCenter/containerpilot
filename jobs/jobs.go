package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
)

// Some magic numbers used internally by restart limits
const (
	unlimited       = -1
	eventBufferSize = 1000
)

// JobStatus is an enum of job health status
type JobStatus int

// JobStatus enum
const (
	statusIdle JobStatus = iota // will be default value before starting
	statusUnknown
	statusHealthy
	statusUnhealthy
	statusMaintenance
)

func (i JobStatus) String() string {
	switch i {
	case 2:
		return "healthy"
	case 3:
		return "unhealthy"
	case 4:
		return "maintenance"
	default:
		// both idle and unknown return unknown for purposes of serialization
		return "unknown"
	}
}

// Job manages the state of a job and its start/stop conditions
type Job struct {
	Name string
	exec *commands.Command

	// service health and discovery
	Status          JobStatus
	statusLock      *sync.RWMutex
	Service         *discovery.ServiceDefinition
	healthCheckExec *commands.Command
	healthCheckName string

	// starting events
	startEvent   events.Event
	startTimeout time.Duration
	startsRemain int

	// stopping events
	stoppingWaitEvent events.Event
	stoppingTimeout   time.Duration

	// timing and restarts
	heartbeat      time.Duration
	restartLimit   int
	restartsRemain int
	frequency      time.Duration

	events.EventHandler // Event handling
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
	job.Rx = make(chan events.Event, eventBufferSize)
	job.statusLock = &sync.RWMutex{}
	if job.Name == "containerpilot" {
		// right now this hardcodes the telemetry service to
		// be always "healthy", but maybe we want to have it verify itself
		// before heartbeating in the future?
		job.setStatus(statusHealthy)
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

// GetStatus returns the current health status of the Job
func (job *Job) GetStatus() JobStatus {
	job.statusLock.RLock()
	defer job.statusLock.RUnlock()
	return job.Status
}

func (job *Job) setStatus(status JobStatus) {
	job.statusLock.Lock()
	defer job.statusLock.Unlock()
	job.Status = status
}

// MarkForMaintenance marks this Job's service for maintenance
func (job *Job) MarkForMaintenance() {
	job.setStatus(statusMaintenance)
	if job.Service != nil {
		job.Service.MarkForMaintenance()
	}
}

// Deregister will deregister this instance of Job's service
func (job *Job) Deregister() {
	if job.Service != nil {
		job.Service.Deregister()
	}
}

// HealthCheck runs the Job's health check executable
func (job *Job) HealthCheck(ctx context.Context) {
	if job.healthCheckExec != nil {
		job.healthCheckExec.Run(ctx, job.Bus)
	}
}

// StartJob runs the Job's executable
func (job *Job) StartJob(ctx context.Context) {
	if job.exec != nil {
		job.exec.Run(ctx, job.Bus)
	}
}

// Kill sends SIGTERM to the Job's executable, if any
func (job *Job) Kill() {
	if job.exec != nil {
		if job.exec.Cmd != nil {
			if job.exec.Cmd.Process != nil {
				job.exec.Cmd.Process.Kill()
			}
		}
	}
}

// Run executes the event loop for the Job
func (job *Job) Run(bus *events.EventBus) {
	job.Subscribe(bus)
	job.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())

	if job.frequency > 0 {
		events.NewEventTimer(ctx, job.Rx, job.frequency,
			fmt.Sprintf("%s.run-every", job.Name))
	}
	if job.heartbeat > 0 {
		events.NewEventTimer(ctx, job.Rx, job.heartbeat,
			fmt.Sprintf("%s.heartbeat", job.Name))
	}
	if job.startTimeout > 0 {
		events.NewEventTimeout(ctx, job.Rx, job.startTimeout,
			fmt.Sprintf("%s.wait-timeout", job.Name))
	}

	go func() {
		defer job.cleanup(ctx, cancel)
		for {
			select {
			case event, ok := <-job.Rx:
				if !ok {
					return
				}
				if job.processEvent(ctx, event) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (job *Job) processEvent(ctx context.Context, event events.Event) bool {
	runEverySource := fmt.Sprintf("%s.run-every", job.Name)
	heartbeatSource := fmt.Sprintf("%s.heartbeat", job.Name)
	startTimeoutSource := fmt.Sprintf("%s.wait-timeout", job.Name)
	var healthCheckName string
	if job.healthCheckExec != nil {
		healthCheckName = job.healthCheckExec.Name
	}

	switch event {
	case events.Event{events.TimerExpired, heartbeatSource}:
		status := job.GetStatus()
		if status != statusMaintenance && status != statusIdle {
			if job.healthCheckExec != nil {
				job.HealthCheck(ctx)
			} else if job.Service != nil {
				// this is the case for non-checked but advertised
				// services like the telemetry endpoint
				job.SendHeartbeat()
			}
		}
	case events.Event{events.TimerExpired, startTimeoutSource}:
		job.Bus.Publish(events.Event{
			Code: events.TimerExpired, Source: job.Name})
		job.Rx <- events.Event{Code: events.Quit, Source: job.Name}
	case events.Event{events.TimerExpired, runEverySource}:
		if !job.restartPermitted() {
			log.Debugf("interval expired but restart not permitted: %v",
				job.Name)
			return true
		}
		job.restartsRemain--
		job.StartJob(ctx)
	case events.Event{events.ExitFailed, healthCheckName}:
		if job.GetStatus() != statusMaintenance {
			job.setStatus(statusUnhealthy)
			job.Bus.Publish(events.Event{events.StatusUnhealthy, job.Name})
		}
	case events.Event{events.ExitSuccess, healthCheckName}:
		if job.GetStatus() != statusMaintenance {
			job.setStatus(statusHealthy)
			job.Bus.Publish(events.Event{events.StatusHealthy, job.Name})
			job.SendHeartbeat()
		}
	case
		events.Event{events.Quit, job.Name},
		events.QuitByClose,
		events.GlobalShutdown:
		if (job.startEvent.Code == events.Stopping ||
			job.startEvent.Code == events.Stopped) &&
			job.exec != nil {
			// "pre-stop" and "post-stop" style jobs ignore the global
			// shutdown and return on their ExitSuccess/ExitFailed.
			// if the stop timeout on the global shutdown is exceeded
			// the whole process gets SIGKILL
			break
		}
		return true
	case events.GlobalEnterMaintenance:
		job.MarkForMaintenance()
	case events.GlobalExitMaintenance:
		job.setStatus(statusUnknown)
	case
		events.Event{events.ExitSuccess, job.Name},
		events.Event{events.ExitFailed, job.Name}:
		if job.frequency > 0 {
			break // periodic jobs ignore previous events
		}
		if job.restartPermitted() {
			job.restartsRemain--
			job.StartJob(ctx)
			break
		}
		if job.startsRemain != 0 {
			break
		}
		log.Debugf("job exited but restart not permitted: %v", job.Name)
		return true
	case job.startEvent:
		if job.startsRemain == 0 {
			return true
		}
		if job.startsRemain != unlimited {
			// if we have unlimited restarts we want to make sure we don't
			// decrement forever and then wrap-around
			job.startsRemain--
			if job.startsRemain == 0 && job.restartsRemain == 0 {
				// prevent ourselves from receiving the start event again
				// if it fires while we're still running the job's exec
				job.startEvent = events.NonEvent
			}
		}
		job.setStatus(statusUnknown)
		job.StartJob(ctx)
	}
	return false
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
	job.Bus.Publish(events.Event{Code: events.Stopping, Source: job.Name})
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
			case events.Event{events.Stopping, stoppingTimeout}:
				break loop
			}
		}
	}
	cancel()
	job.exec.CloseLogs()
	job.Deregister()         // deregister from Consul
	job.Unsubscribe(job.Bus) // deregister from events
	job.Bus.Publish(events.Event{Code: events.Stopped, Source: job.Name})
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (job *Job) String() string {
	return "jobs.Job[" + job.Name + "]"
}
