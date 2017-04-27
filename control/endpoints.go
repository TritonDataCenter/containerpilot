package control

import (
	"encoding/json"
	"net/http"
	"os"
	"io"

	log "github.com/Sirupsen/logrus"
)

// Endpoints defines bridge data that cross the App and HTTPServer API boundary
type Endpoints struct {
	app App
}

// EndpointFunc is an adapter which allows a normal function to serve itself
// over HTTP as a handler. Also allows us to pass through App state.
type EndpointFunc func(http.ResponseWriter, *http.Request)

// ServeHTTP by calling an EndpointFunc that implements some API endpoint
// behavior.
func (ef EndpointFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ef(w, r)
}

// GetEnvHandler generates HTTP response as a test endpoint
func (e Endpoints) GetEnv(w http.ResponseWriter, r *http.Request) {
	log.Debugf("control: Received request at '%s %s'", r.URL, r.Method)

	if r.Method != http.MethodGet {
		failedStatus := http.StatusNotImplemented
		log.Errorf("%s requires GET, not %s", r.URL, r.Method)
		http.Error(w, http.StatusText(failedStatus), failedStatus)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	envJSON, err := json.Marshal(os.Environ())
	if err != nil {
		failedStatus := http.StatusUnprocessableEntity
		log.Errorf("'GET %v' JSON response unprocessable due to error: %v", r.URL, err)
		http.Error(w, http.StatusText(failedStatus), failedStatus)
	}

	log.Debugf("marshaled environ: %v", string(envJSON))
	w.Write(envJSON)
}

// PostReloadHandler reloads ContainerPilot process
func (e Endpoints) PostReload(w http.ResponseWriter, r *http.Request) {
	log.Debugf("control: Received request at '%s %s'", r.URL, r.Method)

	if r.Method != http.MethodPost {
		failedStatus := http.StatusNotImplemented
		log.Errorf("%s requires POST, not %s", r.URL, r.Method)
		http.Error(w, http.StatusText(failedStatus), failedStatus)
		return
	}

	e.app.Reload()
	log.Debug("reloaded app via control plane")

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, "\n")
}
