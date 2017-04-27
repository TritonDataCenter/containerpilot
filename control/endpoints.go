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

// EndpointFunc is an adapter which allows a normal function to serve itself and
// handle incoming HTTP requests. Also allows us to pass through App state in an
// organized fashion.
type EndpointFunc func(http.ResponseWriter, *http.Request)

// ServeHTTP implements intermediate endpoint behavior before calling one of the
// actual handler implementations below.
func (ef EndpointFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("control: '%s %s' requested", r.Method, r.URL)
	ef(w, r)
}

// GetEnv generates HTTP response which returns current OS environ. Used as a
// test endpoint.
func (e Endpoints) GetEnviron(w http.ResponseWriter, r *http.Request) {
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

// PostReload reloads ContainerPilot process configuration and generates null
// HTTP response.
func (e Endpoints) PostReload(w http.ResponseWriter, r *http.Request) {
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
