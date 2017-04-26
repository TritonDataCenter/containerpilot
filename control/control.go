package control

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
)

// SocketType is the default listener type
var SocketType = "unix"

// HTTPServer contains the state of the HTTP Server used by ContainerPilot's
// HTTP transport control plane. Currently this is listening via a UNIX socket
// file.
type HTTPServer struct {
	http.Server
	mux         *http.ServeMux
	Addr        string
	lock        sync.RWMutex
}

// App interface ensures App object passed into Start contains appropriate
// methods for use by control endpoint handlers.
type App interface {
	Reload()
	ToggleMaintenanceMode()
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

// Start starts serving HTTP over the control server
func (s *HTTPServer) Start(app App) {
	s.lock.Lock()
	defer s.lock.Unlock()

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
func (s *HTTPServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Warn("control: failed to shutdown HTTP control plane")
		return err
	}

	log.Debug("control: shutdown HTTP control plane")
	return nil
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
