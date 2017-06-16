package telemetry

import (
	"encoding/json"
	"net/http"

	"github.com/joyent/containerpilot/jobs"
	"github.com/joyent/containerpilot/watches"
)

// StatusHandler implements http.Handler
type StatusHandler struct {
	t        *Telemetry
	response *statusResponse
}

// NewStatusHandler ...
func NewStatusHandler(t *Telemetry) StatusHandler {
	return StatusHandler{t: t}
}

func (hand StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		failedStatus := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(failedStatus), failedStatus)
		return
	}
	if hand.response == nil {
		response := &statusResponse{}
		for _, job := range hand.t.Jobs {
			if job.Service.Port != 0 {
				service := jobStatusResponse{
					name:    job.Name,
					address: job.Service.IPAddress,
					port:    job.Service.Port,
					status:  job.GetStatus(),
				}
				response.services = append(response.services, service)
			}
		}
		for _, watch := range hand.t.Watches {
			response.watches = append(response.watches, watch.Name)
		}
		hand.response = response
	} else {
		for _, job := range hand.t.Jobs {
			status := job.GetStatus()
			for _, service := range hand.response.services {
				if service.name == job.Name {
					service.status = status
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(hand.response)
}

type statusResponse struct {
	services []jobStatusResponse
	watches  []string
}

type jobStatusResponse struct {
	name    string
	address string
	port    int
	status  jobs.JobStatus
}

// MonitorJobs ... (TODO: has a bad name)
func (t *Telemetry) MonitorJobs(jobs []*jobs.Job) {
	if t != nil {
		t.Jobs = jobs
	}
}

// MonitorWatches ... (TODO: has a bad name)
func (t *Telemetry) MonitorWatches(watches []*watches.Watch) {
	if t != nil {
		t.Watches = watches
	}
}
