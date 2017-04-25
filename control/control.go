package control

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
//	"sync"

	log "github.com/Sirupsen/logrus"
)

var SocketType string = "unix"

// HTTPServer contains the state of the HTTP Server used by ContainerPilot's
// HTTP transport control plane. Currently this is listening via a UNIX socket
// file.
type HTTPServer struct {
	http.Server
	mux  *http.ServeMux
	Addr string
	// lock       sync.RWMutex
}

// NewHTTPServer initializes a new control server for manipulating
// ContainerPilot's runtime configuration.
func NewHTTPServer(cfg *Config) (*HTTPServer, error) {
	if cfg == nil {
		err := errors.New("control server not loading due to missing config")
		return nil, err
	}

	mux := http.NewServeMux()

	return &HTTPServer{
		Addr: cfg.SocketPath,
		mux: mux,
	}, nil
}

// Serve starts serving the control server
func (s *HTTPServer) Start(app interface{}) {
	// s.lock.Lock()
	// defer s.lock.Unlock()

	s.mux.HandleFunc("/v3/env", s.getEnvHandler)
	s.mux.HandleFunc("/v3/reload", s.postReloadHandler)

	s.Handler = s.mux

	ln, err := net.Listen(SocketType, s.Addr)
	if err != nil {
		log.Fatalf("error serving socket at %s: %v", s.Addr, err)
	}

	go func() {
		log.Infof("control: Serving at %s", s.Addr)
		log.Fatal(s.Serve(ln))
		log.Debugf("control: Stopped serving at %s", s.Addr)
	}()
}

// Stop shuts down the control server gracefully
func (s *HTTPServer) Stop() {
	if err := s.Shutdown(nil); err != nil {
		log.Fatal("control: failed to shutdown HTTP control plane")
	} else {
		log.Debug("control: shutdown HTTP control plane")
	}
}

// getEnvHandler generates HTTP response as a test endpoint
func (s *HTTPServer) getEnvHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		failedStatus := http.StatusNotImplemented
		log.Errorf("'GET %v' not responding to request method '%v'", r.URL, r.Method)
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

// postReloadHandler reloads ContainerPilot process
func (s *HTTPServer) postReloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		failedStatus := http.StatusNotImplemented
		log.Errorf("'POST %v' not responding to request method '%v'", r.URL, r.Method)
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
