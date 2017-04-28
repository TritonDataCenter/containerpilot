package control

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
)

// Endpoints defines bridge data that cross the App and HTTPServer API boundary
type Endpoints struct {
	app App
}

// PostHandler is an adapter which allows a normal function to serve itself and
// handle incoming HTTP POST requests. Also allows us to pass through App state
// in a more organized fashion.
type PostHandler func(http.ResponseWriter, *http.Request)

// ServeHTTP implements intermediate endpoint behavior for POST
// requests. Subsequently calls actual handler implementations defined below.
func (ph PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("control: '%s %s' requested", r.Method, r.URL)

	if r.Method != http.MethodPost {
		failedStatus := http.StatusNotImplemented
		log.Errorf("%s requires POST, not %s", r.URL, r.Method)
		http.Error(w, http.StatusText(failedStatus), failedStatus)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	ph(w, r)

	io.WriteString(w, "\n")
}

// PutEnviron handles incoming HTTP POST requests containing JSON environment
// variables and updates the environment of our current ContainerPilot
// process. Returns null HTTP response.
func (e Endpoints) PutEnviron(w http.ResponseWriter, r *http.Request) {
	var postEnv map[string]string

	errFunc := func(err error) {
		failedStatus := http.StatusUnprocessableEntity
		log.Errorf("'%v %v' request unprocessable due to error:\n%v", r.Method, r.URL, err)
		http.Error(w, http.StatusText(failedStatus), failedStatus)
	}

	jsonBlob, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		errFunc(err)
	}

	err = json.Unmarshal(jsonBlob, &postEnv)
	if err != nil {
		errFunc(err)
	}

	for envKey, envValue := range postEnv {
		os.Setenv(envKey, envValue)
	}
}

// PostReload handles incoming HTTP POST requests and reloads our current
// ContainerPilot process configuration. Returns null HTTP response.
func (e Endpoints) PostReload(w http.ResponseWriter, r *http.Request) {
	e.app.Reload()
	log.Debug("control: reloaded app via control plane")
}
