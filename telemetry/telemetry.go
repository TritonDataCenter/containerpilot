// Package telemetry provides a Prometheus client and the configuration
// for metrics collectors
package telemetry

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/joyent/containerpilot/version"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Metrics.
type Telemetry struct {
	Metrics []*Metric // supports '/metrics' endpoint fields
	Status  *Status   // supports '/status' endpoint fields

	// server
	router *http.ServeMux
	addr   net.TCPAddr

	http.Server
}

// NewTelemetry configures a new prometheus Telemetry server
func NewTelemetry(cfg *Config) *Telemetry {
	if cfg == nil {
		return nil
	}
	t := &Telemetry{
		Metrics: []*Metric{},
		Status:  &Status{Version: version.Version},
	}
	t.addr = cfg.addr

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.Handler())
	router.Handle("/status", NewStatusHandler(t))
	t.Handler = router

	for _, sensorCfg := range cfg.MetricConfigs {
		sensor := NewMetric(sensorCfg)
		t.Metrics = append(t.Metrics, sensor)
	}

	return t
}

// Run executes the event loop for the telemetry server
func (t *Telemetry) Run(ctx context.Context) {
	t.Start()
	go func() {
		defer t.Stop(ctx)
		<-ctx.Done()
		return
	}()
}

// Start starts serving the telemetry service
func (t *Telemetry) Start() {
	ln := t.listenWithRetry()
	go func() {
		log.Infof("telemetry: serving at %s", t.addr.String())
		t.Serve(ln)
		log.Debugf("telemetry: stopped serving at %s", t.addr.String())
	}()
}

// on a reload we can't guarantee that the control server will be shut down
// and the socket file cleaned up before we're ready to start again, so we'll
// retry with the listener a few times before bailing out.
func (t *Telemetry) listenWithRetry() net.Listener {
	var (
		err error
		ln  net.Listener
	)
	for i := 0; i < 10; i++ {
		ln, err = net.Listen(t.addr.Network(), t.addr.String())
		if err == nil {
			return ln
		}
		time.Sleep(time.Second)
	}
	log.Fatalf("error listening to socket at %s: %v", t.addr.String(), err)
	return nil
}

// Stop shuts down the telemetry service
func (t *Telemetry) Stop(pctx context.Context) {
	log.Debug("telemetry: stopping server")
	ctx, cancel := context.WithCancel(pctx)
	defer cancel()
	if err := t.Shutdown(ctx); err != nil {
		log.Warnf("telemetry: failed to gracefully shutdown server: %v", err)
		return
	}
	log.Debug("telemetry: completed graceful shutdown of server")
}
