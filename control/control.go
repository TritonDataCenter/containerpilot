// Package control provides a HTTP server listening on the unix domain
// socket for use as a control plane, as well as all the HTTP endpoints.
package control

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/tritondatacenter/containerpilot/events"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// SocketType is the default listener type
var (
	SocketType     = "unix"
	ErrMissingAddr = errors.New("control server not loading due to missing config")
)

var collector *prometheus.CounterVec

func init() {
	collector = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "containerpilot_control_http_requests",
		Help: "count of requests to control socket, partitioned by path and HTTP code",
	}, []string{"code", "path"})
	prometheus.MustRegister(collector)
}

// HTTPServer contains the state of the HTTP Server used by ContainerPilot's
// HTTP transport control plane. Currently this is listening via a UNIX socket
// file.
type HTTPServer struct {
	Addr string
	Bus  *events.EventBus

	http.Server
	events.Publisher
}

// NewHTTPServer initializes a new control server for manipulating
// ContainerPilot's runtime configuration.
func NewHTTPServer(cfg *Config) (*HTTPServer, error) {
	srv := &HTTPServer{
		Addr: cfg.SocketPath,
	}
	if err := srv.Validate(); err != nil {
		return nil, fmt.Errorf("control: validate failed with %s", err)
	}

	return srv, nil
}

// Validate validates the state of the control server and ensures that the
// socket does not exist prior to setting up the listener (bind).
func (srv *HTTPServer) Validate() error {
	if srv.Addr == "" {
		return ErrMissingAddr
	}
	if _, err := os.Stat(srv.Addr); err == nil {
		log.Debugf("control: unlinking previous socket at %s", srv.Addr)
		if err := os.Remove(srv.Addr); err != nil {
			return err
		}
	}

	return nil
}

// Run executes the event loop for the control server
func (srv *HTTPServer) Run(pctx context.Context, bus *events.EventBus) {
	ctx, cancel := context.WithCancel(pctx)
	srv.Register(bus)
	srv.Start(cancel)

	go func() {
		defer srv.Stop()
		<-ctx.Done()
	}()
}

// Start sets up API routes with the event bus, listens on the control socket,
// and serves the HTTP server.
func (srv *HTTPServer) Start(cancel context.CancelFunc) {
	endpoints := &Endpoints{
		bus:    srv.Publisher.Bus,
		cancel: cancel,
	}

	router := http.NewServeMux()
	router.Handle("/v3/environ",
		PostHandler(endpoints.PutEnviron))
	router.Handle("/v3/reload",
		PostHandler(endpoints.PostReload))
	router.Handle("/v3/metric",
		PostHandler(endpoints.PostMetric))
	router.Handle("/v3/maintenance/enable",
		PostHandler(endpoints.PostEnableMaintenanceMode))
	router.Handle("/v3/maintenance/disable",
		PostHandler(endpoints.PostDisableMaintenanceMode))
	router.HandleFunc("/v3/ping", GetPing)

	srv.Handler = router
	srv.SetKeepAlivesEnabled(false)
	log.Debug("control: initialized router for control server")

	ln := srv.listenWithRetry()

	go func() {
		log.Infof("control: serving at %s", srv.Addr)
		srv.Serve(ln)
		log.Debugf("control: stopped serving at %s", srv.Addr)
	}()
}

// on a reload we can't guarantee that the control server will be shut down and
// the socket file cleaned up before we're ready to start again, so we'll retry
// with the listener a few times before bailing out.
func (srv *HTTPServer) listenWithRetry() net.Listener {
	var (
		err error
		ln  net.Listener
	)
	for i := 0; i < 10; i++ {
		ln, err = net.Listen(SocketType, srv.Addr)
		if err == nil {
			log.Debugf("control: listening to %s", srv.Addr)
			return ln
		}
		time.Sleep(time.Second)
	}
	log.Fatalf("error listening to socket at %s: %v", srv.Addr, err)
	return nil
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

	srv.Unregister()
	log.Debug("control: completed graceful shutdown of control server")
	return nil
}
