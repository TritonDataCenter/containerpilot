package control

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/tritondatacenter/containerpilot/events"
	log "github.com/sirupsen/logrus"
)

// Endpoints wraps the EventBus so we can bridge data across the App and
// HTTPServer API boundary
type Endpoints struct {
	bus    *events.EventBus
	cancel context.CancelFunc
}

// PostHandler is an adapter which allows a normal function to serve itself and
// handle incoming HTTP POST requests, and allows us to pass thru EventBus to
// handlers
type PostHandler func(*http.Request) (interface{}, int)

func (pw PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		failedStatus := http.StatusMethodNotAllowed
		http.Error(w, http.StatusText(failedStatus), failedStatus)
		collector.WithLabelValues(
			strconv.Itoa(http.StatusMethodNotAllowed), r.URL.Path).Inc()
		return
	}
	resp, status := pw(r)
	switch status {
	case http.StatusOK:
		if resp != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		} else {
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, "\n")
		}
	default:
		http.Error(w, http.StatusText(status), status)
	}
	collector.WithLabelValues(strconv.Itoa(status), r.URL.Path).Inc()
}

// PutEnviron handles incoming HTTP POST requests containing JSON environment
// variables and updates the environment of our current ContainerPilot
// process. Returns empty response or HTTP422.
func (e Endpoints) PutEnviron(r *http.Request) (interface{}, int) {
	var postEnv map[string]string
	jsonBlob, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return nil, http.StatusUnprocessableEntity
	}
	err = json.Unmarshal(jsonBlob, &postEnv)
	if err != nil {
		return nil, http.StatusUnprocessableEntity
	}
	for envKey, envValue := range postEnv {
		os.Setenv(envKey, envValue)
	}
	return nil, http.StatusOK
}

// PostReload handles incoming HTTP POST requests and reloads our current
// ContainerPilot process configuration.  Returns empty response or HTTP422.
func (e Endpoints) PostReload(r *http.Request) (interface{}, int) {
	defer e.cancel()
	log.Debug("control: reloading app via control plane")
	if r.Body != nil {
		defer r.Body.Close()
	}
	e.bus.SetReloadFlag()
	e.bus.Shutdown()
	log.Debug("control: reloaded app via control plane")
	return nil, http.StatusOK
}

// PostEnableMaintenanceMode handles incoming HTTP POST requests and toggles
// ContainerPilot maintenance mode on. Returns empty response or HTTP422.
func (e Endpoints) PostEnableMaintenanceMode(r *http.Request) (interface{}, int) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	e.bus.Publish(events.GlobalEnterMaintenance)
	return nil, http.StatusOK
}

// PostDisableMaintenanceMode handles incoming HTTP POST requests and toggles
// ContainerPilot maintenance mode on. Returns empty response or HTTP422.
func (e Endpoints) PostDisableMaintenanceMode(r *http.Request) (interface{}, int) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	e.bus.Publish(events.GlobalExitMaintenance)
	return nil, http.StatusOK
}

// PostMetric handles incoming HTTP POST requests, serializes the metrics
// into Events, and publishes them for sensors to record their values.
// Returns empty response or HTTP422.
func (e Endpoints) PostMetric(r *http.Request) (interface{}, int) {
	var postMetrics map[string]interface{}
	jsonBlob, err := io.ReadAll(r.Body)

	defer r.Body.Close()
	if err != nil {
		return nil, http.StatusUnprocessableEntity
	}
	err = json.Unmarshal(jsonBlob, &postMetrics)
	if err != nil {
		log.Debug(err)
		return nil, http.StatusUnprocessableEntity
	}
	for metricKey, metricValue := range postMetrics {
		eventVal := fmt.Sprintf("%v|%v", metricKey, metricValue)
		e.bus.Publish(events.Event{Code: events.Metric, Source: eventVal})
	}
	return nil, http.StatusOK
}

// GetPing allows us to check if the control socket is up without
// making a mutation of ContainerPilot's state
func GetPing(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.WriteHeader(http.StatusOK)
	collector.WithLabelValues("200", r.URL.Path).Inc()
	io.WriteString(w, "\n")
}
