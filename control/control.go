package control

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
)

const (
	// SocketType is the default listener type
	SocketType = "unix"
)

// HTTPServer contains the state of the HTTP Server used by ContainerPilot's
// HTTP transport control plane. Currently this is listening via a UNIX socket
// file.
type HTTPServer struct {
	mux        *http.ServeMux
	addr       net.UnixAddr
	listening  bool
	lock       sync.RWMutex
}

// NewHTTPServer initializes a new control server for manipulating
// ContainerPilot's runtime configuration.
func NewHTTPServer(cfg *Config) (*HTTPServer, error) {
	if cfg == nil {
		err := errors.New("Control server not loaded due to missing config")
		return nil, err
	}

	mux := http.NewServeMux()
	addr := net.UnixAddr{
		Name: cfg.SocketPath,
		Net: SocketType,
	}

	return &HTTPServer{
		mux: mux,
		addr: addr,
		listening: false,
	}, nil
}

var listener net.Listener

// Serve starts serving the control server
func (s *HTTPServer) Serve() {
	s.lock.Lock()
	defer s.lock.Unlock()

	// ref https://github.com/joyent/containerpilot/pull/165
	if listener != nil {
		return
	}

	s.mux.HandleFunc("/env", s.getEnvHandler)

	ln, err := net.Listen(s.addr.Network(), s.addr.String())
	if err != nil {
		log.Fatalf("Error serving socket at %s: %v", s.addr.String(), err)
	}

	listener = ln
	s.listening = true

	go func() {
		log.Infof("control: Serving at %s", s.addr.String())
		log.Fatal(http.Serve(ln, s.mux))
		log.Debugf("control: Stopped serving at %s", s.addr.String())
	}()
}

// Shutdown shuts down the control server
func (s *HTTPServer) Shutdown() {
	s.listening = false
	log.Debug("control: Shutdown received but currently a no-op")
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

	log.Debugf("Marshaled environ: %v", string(envJSON))
	w.Write(envJSON)
}
