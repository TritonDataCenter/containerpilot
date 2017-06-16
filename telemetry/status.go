package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
			if job.Service != nil && job.Service.Port != 0 {
				service := jobStatusResponse{
					Name:    job.Name,
					Address: job.Service.IPAddress,
					Port:    job.Service.Port,
					Status:  fmt.Sprintf("%s", job.GetStatus()),
				}
				response.Services = append(response.Services, service)
			}
		}
		for _, watch := range hand.t.Watches {
			name := strings.TrimPrefix(watch.Name, "watch.")
			response.Watches = append(response.Watches, name)
		}
		hand.response = response
	} else {
		for _, job := range hand.t.Jobs {
			status := fmt.Sprintf("%s", job.GetStatus())
			for _, service := range hand.response.Services {
				if service.Name == job.Name {
					service.Status = status
				}
			}
		}
	}
	fmt.Printf("%+v\n", hand.response)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(hand.response)
	if err != nil {
		fmt.Println(err)
	}
}

type statusResponse struct {
	Services []jobStatusResponse
	Watches  []string
}

type jobStatusResponse struct {
	Name    string
	Address string
	Port    int
	Status  string
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
