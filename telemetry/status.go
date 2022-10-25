package telemetry

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tritondatacenter/containerpilot/jobs"
	"github.com/tritondatacenter/containerpilot/watches"
)

// Status persists all the data the telemetry server needs to serve
// the '/status' endpoint
type Status struct {
	Version  string
	jobs     []*jobs.Job
	Jobs     []*jobStatusResponse
	Services []*serviceStatusResponse
	Watches  []string
}

type jobStatusResponse struct {
	Name   string
	Status string
}

type serviceStatusResponse struct {
	Name    string
	Address string
	Port    int
	Status  string
}

// StatusHandler implements http.Handler
type StatusHandler struct {
	telem *Telemetry
}

// NewStatusHandler constructs a StatusHandler with a pointer
// to the Telemetry server
func NewStatusHandler(t *Telemetry) StatusHandler {
	return StatusHandler{telem: t}
}

// ServeHTTP implements http.Handler for StatusHandler
func (sh StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		failedStatus := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(failedStatus), failedStatus)
		return
	}
	for _, job := range sh.telem.Status.jobs {
		status := job.GetStatus().String()
		for _, service := range sh.telem.Status.Services {
			if service.Name == job.Name {
				service.Status = status
			}
		}
		for _, jobStatus := range sh.telem.Status.Jobs {
			if jobStatus.Name == job.Name {
				jobStatus.Status = status
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(sh.telem.Status)
}

// MonitorJobs adds a list of Jobs for the /status handler to monitor
func (t *Telemetry) MonitorJobs(jobs []*jobs.Job) {
	if t != nil {
		for _, job := range jobs {
			t.Status.jobs = append(t.Status.jobs, job)
			if job.Service != nil && job.Service.Port != 0 {
				serviceResponse := &serviceStatusResponse{
					Name:    job.Name,
					Address: job.Service.IPAddress,
					Port:    job.Service.Port,
					Status:  job.GetStatus().String(),
				}
				t.Status.Services = append(t.Status.Services, serviceResponse)
			} else {
				jobResponse := &jobStatusResponse{
					Name:   job.Name,
					Status: job.GetStatus().String(),
				}
				t.Status.Jobs = append(t.Status.Jobs, jobResponse)
			}
		}
	}
}

// MonitorWatches adds a list of Watches for the /status handler to monitor
func (t *Telemetry) MonitorWatches(watches []*watches.Watch) {

	// these watch names are cached forever because they don't change
	// unless we reload ContainerPilot itself (and this server)
	if t != nil {
		for _, watch := range watches {
			name := strings.TrimPrefix(watch.Name, "watch.")
			t.Status.Watches = append(t.Status.Watches, name)
		}
	}
}
