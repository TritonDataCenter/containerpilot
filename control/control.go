package control

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
)

// SocketType is the default listener type
var SocketType = "unix"

// HTTPServer contains the state of the HTTP Server used by ContainerPilot's
// HTTP transport control plane. Currently this is listening via a UNIX socket
// file.
type HTTPServer struct {
	http.Server
	Addr                string
	events.EventHandler // Event handling
}

// NewHTTPServer initializes a new control server for manipulating
// ContainerPilot's runtime configuration.
func NewHTTPServer(cfg *Config) (*HTTPServer, error) {
	if cfg == nil {
		err := errors.New("control server not loading due to missing config")
		return nil, err
	}
	srv := &HTTPServer{
		Addr: cfg.SocketPath,
	}
	srv.Rx = make(chan events.Event, 10)
	srv.Flush = make(chan bool)
	return srv, nil
}

// Run executes the event loop for the control server
func (srv *HTTPServer) Run(bus *events.EventBus) {
	srv.Subscribe(bus)
	srv.Bus = bus
	srv.Start()

	go func() {
	loop:
		for {
			event := <-srv.Rx
			switch event {
			case
				events.QuitByClose,
				events.GlobalShutdown:
				break loop
			}
		}
		srv.Stop()
	}()
}

// Start sets up API routes with the event bus, listens on the control
// socket, and serves the HTTP server.
func (srv *HTTPServer) Start() {
	endpoints := &Endpoints{srv.Bus}

	router := http.NewServeMux()
	router.Handle("/v3/environ", PostHandler(endpoints.PutEnviron))
	router.Handle("/v3/reload", PostHandler(endpoints.PostReload))
	srv.Handler = router

	srv.SetKeepAlivesEnabled(false)
	log.Debug("control: initialized router for control server")

	ln, err := net.Listen(SocketType, srv.Addr)
	if err != nil {
		log.Fatalf("error listening to socket at %s: %v", srv.Addr, err)
	}

	go func() {
		log.Infof("control: serving at %s", srv.Addr)
		srv.Serve(ln)
		log.Debugf("control: stopped serving at %s", srv.Addr)
	}()

}

// Stop shuts down the control server gracefully
func (srv *HTTPServer) Stop() error {
	// This timeout won't stop the configuration reload process, since that
	// happens async, but timing out can pre-emptively close the HTTP connection
	// that fired the reload in the first place. If pre-emptive timeout occurs
	// than CP only throws a warning in its logs.
	//
	// Also, 600 seemed to be the magic number... I'm sure it'll vary.
	log.Debug("control: stopping control server")
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()
	defer os.Remove(srv.Addr)
	if err := srv.Shutdown(ctx); err != nil {
		log.Warnf("control: failed to gracefully shutdown control server: %v", err)
		return err
	}

	srv.Unsubscribe(srv.Bus)
	close(srv.Rx)
	log.Debug("control: completed graceful shutdown of control server")
	return nil
}
