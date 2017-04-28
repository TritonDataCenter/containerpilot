package control

import (
	"context"
	"errors"
	"net"
	"net/http"
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

	return &HTTPServer{
		Addr: cfg.SocketPath,
	}, nil
}

// Start sets up API routes, passing along App state, listens on the control
// socket, and serves the HTTP server.
func (s *HTTPServer) Start(app App) {
	s.lock.Lock()
	defer s.lock.Unlock()

	endpoints := &Endpoints{app}

	router := http.NewServeMux()
	router.Handle("/v3/environ", PostHandler(endpoints.PutEnviron))
	router.Handle("/v3/reload", PostHandler(endpoints.PostReload))
	s.Handler = router

	s.SetKeepAlivesEnabled(false)

	log.Debug("control: Initialized router for control server")

	ln, err := net.Listen(SocketType, s.Addr)
	if err != nil {
		log.Fatalf("error listening to socket at %s: %v", s.Addr, err)
	}

	go func() {
		log.Infof("control: Serving at %s", s.Addr)
		s.Serve(ln)
		log.Debugf("control: Stopped serving at %s", s.Addr)
	}()
}

// Stop shuts down the control server gracefully
func (s *HTTPServer) Stop() error {
	// This timeout won't stop the configuration reload process, since that
	// happens async, but timing out can pre-emptively close the HTTP connection
	// that fired the reload in the first place. If pre-emptive timeout occurs
	// than CP only throws a warning in it's logs.
	//
	// Also, 600 seemed to be the magic number... I'm sure it'll vary.
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Warn("control: failed to gracefully shutdown control server")
		return err
	}

	log.Debug("control: completed graceful shutdown of control server")
	return nil
}
