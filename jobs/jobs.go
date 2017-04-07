package jobs

import (
	"context"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/events"
)

// Some magic numbers used internally by restart limits
const (
	unlimitedRestarts = -1
	eventBufferSize   = 1000
)

// Job manages the state of a job and its start/stop conditions
type Job struct {
	Name             string
	exec             *commands.Command
	Status           bool // TODO: we'll need this to carry more info than bool
	discoveryCatalog discovery.Backend
	Definition       *discovery.ServiceDefinition

	// related events
	whenEvent         events.Event
	whenTimeout       time.Duration
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
		Definition:        cfg.definition,
		whenEvent:         cfg.whenEvent,
		whenTimeout:       cfg.whenTimeout,
		stoppingWaitEvent: cfg.stoppingWaitEvent,
		stoppingTimeout:   cfg.stoppingTimeout,
		restartLimit:      cfg.restartLimit,
		restartsRemain:    cfg.restartLimit,
		frequency:         cfg.freqInterval,
	}
	job.Rx = make(chan events.Event, eventBufferSize)
	job.Flush = make(chan bool)
	if job.Name == "containerpilot" {
		// TODO: right now this hardcodes the telemetry service to
		// be always "on", but maybe we want to have it verify itself
		// before heartbeating in the future
		job.Status = true
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
	if job.discoveryCatalog != nil || job.Definition != nil {
		job.discoveryCatalog.SendHeartbeat(job.Definition)
	}
}

// MarkForMaintenance marks this Job's service for maintenance
func (job *Job) MarkForMaintenance() {
	if job.discoveryCatalog != nil || job.Definition != nil {
		job.discoveryCatalog.MarkForMaintenance(job.Definition)
	}
}

// Deregister will deregister this instance of Job's service
func (job *Job) Deregister() {
	if job.discoveryCatalog != nil || job.Definition != nil {
		job.discoveryCatalog.Deregister(job.Definition)
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
	if job.whenTimeout > 0 {
		events.NewEventTimeout(ctx, job.Rx, job.whenTimeout, startTimeoutSource)
	}

	go func() {
	loop: // aw yeah, goto like it's 1968!
		for {
			event := <-job.Rx
			switch event {
			case events.Event{events.TimerExpired, heartbeatSource}:
				// non-advertised jobs shouldn't receive this event
				// but we'll hit a null-pointer if we screw it up
				if job.Status == true && job.Definition != nil {
					job.SendHeartbeat()
				}
			case events.Event{events.TimerExpired, startTimeoutSource}:
				job.Bus.Publish(events.Event{
					Code: events.TimerExpired, Source: job.Name})
				job.Rx <- events.Event{Code: events.Quit, Source: job.Name}
			case events.Event{events.TimerExpired, runEverySource}:
				if !job.restartPermitted() {
					log.Debugf("restart not permitted: %v", job.Name)
					break loop
				}
				job.restartsRemain--
				job.Rx <- job.whenEvent
			case events.Event{events.StatusUnhealthy, job.Name}:
				// TODO v3: add a "SendFailedHeartbeat" method to fail faster
				job.Status = false
			case events.Event{events.StatusHealthy, job.Name}:
				job.Status = true
			case
				events.Event{events.Quit, job.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				break loop
			case
				events.Event{events.ExitSuccess, job.Name},
				events.Event{events.ExitFailed, job.Name}:
				if job.frequency > 0 {
					break // note: breaks switch only
				}
				if !job.restartPermitted() {
					log.Debugf("restart not permitted: %v", job.Name)
					break loop
				}
				job.restartsRemain--
				job.Rx <- job.whenEvent
			case job.whenEvent:
				job.StartJob(ctx)
			}
		}
		job.cleanup(ctx, cancel)
	}()
}

func (job *Job) restartPermitted() bool {
	if job.restartLimit == unlimitedRestarts || job.restartsRemain > 0 {
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
