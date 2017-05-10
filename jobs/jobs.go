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

// note: this num may end up being public so we can use it in the status
// endpoint, but let's leave it as unexported until that API has been decided
type jobStatus int

// jobStatus enum
const (
	statusUnknown jobStatus = iota
	statusHealthy
	statusUnhealthy
	statusMaintenance
)

// Job manages the state of a job and its start/stop conditions
type Job struct {
	Name string
	exec *commands.Command

	// service health and discovery
	Status           jobStatus
	statusLock       *sync.RWMutex
	discoveryCatalog discovery.Backend
	Service          *discovery.ServiceDefinition
	healthCheckExec  *commands.Command
	healthCheckName  string

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
		discoveryCatalog:  cfg.discoveryCatalog,
		Service:           cfg.definition,
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
	job.Flush = make(chan bool, 1)
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
	if job.discoveryCatalog != nil || job.Service != nil {
		job.discoveryCatalog.SendHeartbeat(job.Service)
	}
}

// note: this method may end up being public so we can use it in the status
// endpoint, but let's leave it as unexported until that API has been decided
func (job *Job) getStatus() jobStatus {
	job.statusLock.RLock()
	defer job.statusLock.RUnlock()
	return job.Status
}

func (job *Job) setStatus(status jobStatus) {
	job.statusLock.Lock()
	defer job.statusLock.Unlock()
	job.Status = status
}

// MarkForMaintenance marks this Job's service for maintenance
func (job *Job) MarkForMaintenance() {
	job.setStatus(statusMaintenance)
	if job.discoveryCatalog != nil || job.Service != nil {
		job.discoveryCatalog.MarkForMaintenance(job.Service)
	}
}

// Deregister will deregister this instance of Job's service
func (job *Job) Deregister() {
	if job.discoveryCatalog != nil || job.Service != nil {
		job.discoveryCatalog.Deregister(job.Service)
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

	runEverySource := fmt.Sprintf("%s.run-every", job.Name)
	if job.frequency > 0 {
		events.NewEventTimer(ctx, job.Rx, job.frequency, runEverySource)
	}

	heartbeatSource := fmt.Sprintf("%s.heartbeat", job.Name)
	if job.heartbeat > 0 {
		events.NewEventTimer(ctx, job.Rx, job.heartbeat, heartbeatSource)
	}

	startTimeoutSource := fmt.Sprintf("%s.wait-timeout", job.Name)
	if job.startTimeout > 0 {
		events.NewEventTimeout(ctx, job.Rx, job.startTimeout, startTimeoutSource)
	}

	var healthCheckName string
	if job.healthCheckExec != nil {
		healthCheckName = job.healthCheckExec.Name
	}

	go func() {
	loop: // aw yeah, goto like it's 1968!
		for {
			event := <-job.Rx
			switch event {
			case events.Event{events.TimerExpired, heartbeatSource}:
				if job.getStatus() != statusMaintenance {
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
					break loop
				}
				job.restartsRemain--
				job.StartJob(ctx)
			case events.Event{events.ExitFailed, healthCheckName}:
				job.setStatus(statusUnhealthy)
				job.Bus.Publish(events.Event{events.StatusUnhealthy, job.Name})
				// do we want a "SendFailedHeartbeat" method to fail faster?
			case events.Event{events.ExitSuccess, healthCheckName}:
				job.setStatus(statusHealthy)
				job.Bus.Publish(events.Event{events.StatusHealthy, job.Name})
				job.SendHeartbeat()
			case
				events.Event{events.Quit, job.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				break loop
			case events.GlobalEnterMaintenance:
				job.MarkForMaintenance()
			case events.GlobalExitMaintenance:
				job.setStatus(statusUnknown)
			case
				events.Event{events.ExitSuccess, job.Name},
				events.Event{events.ExitFailed, job.Name}:
				if job.frequency > 0 {
					break // note: breaks switch only
				}
				if !job.restartPermitted() {
					log.Debugf("job exited but restart not permitted: %v",
						job.Name)
					break loop
				}
				job.restartsRemain--
				job.StartJob(ctx)
			case job.startEvent:
				if job.startsRemain == unlimited || job.startsRemain > 0 {
					job.startsRemain--
					job.StartJob(ctx)
				} else {
					break loop
				}
			}
		}
		job.cleanup(ctx, cancel)
	}()
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
	job.Unsubscribe(job.Bus)
	job.Deregister()
	close(job.Rx)
	cancel()
	job.Bus.Publish(events.Event{Code: events.Stopped, Source: job.Name})
	job.exec.CloseLogs()
	job.Flush <- true
}

// String implements the stdlib fmt.Stringer interface for pretty-printing
func (job *Job) String() string {
	return "jobs.Job[" + job.Name + "]"
}
