package telemetry

import (
	"context"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/jobs"
	"github.com/joyent/containerpilot/watches"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Metrics.
type Telemetry struct {
	Metrics   []*Metric
	Jobs      []*jobs.Job
	Watches   []*watches.Watch
	Path      string
	heartbeat time.Duration
	router    *http.ServeMux
	addr      net.TCPAddr

	http.Server
	events.EventHandler // Event handling
}

// NewTelemetry configures a new prometheus Telemetry server
func NewTelemetry(cfg *Config) *Telemetry {
	if cfg == nil {
		return nil
	}
	t := &Telemetry{
		Metrics: []*Metric{},
		Jobs:    []*jobs.Job{},
	}
	t.addr = cfg.addr
	router := http.NewServeMux()
	router.Handle("/metrics", prometheus.Handler())
	router.Handle("/status", NewStatusHandler(t))
	t.Handler = router

	for _, sensorCfg := range cfg.MetricConfigs {
		sensor := NewMetric(sensorCfg)
		t.Metrics = append(t.Metrics, sensor)
	}
	t.Rx = make(chan events.Event, 10)
	return t
}

// Run executes the event loop for the telemetry server
func (t *Telemetry) Run(bus *events.EventBus) {
	t.Subscribe(bus, true)
	t.Bus = bus
	t.Start()

	go func() {
		defer t.Stop()
		for {
			event := <-t.Rx
			switch event {
			case
				events.QuitByClose,
				events.GlobalShutdown:
				return
			}
		}
	}()
}

// Start starts serving the telemetry service
func (t *Telemetry) Start() {
	ln := t.listenWithRetry()
	go func() {
		log.Infof("telemetry: serving at %s", t.Addr)
		t.Serve(ln)
		log.Debugf("telemetry: stopped serving at %s", t.Addr)
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
	log.Fatalf("error listening to socket at %s: %v", t.Addr, err)
	return nil
}

// Stop shuts down the telemetry service
func (t *Telemetry) Stop() {
	log.Debug("telemetry: stopping server")
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := t.Shutdown(ctx); err != nil {
		log.Warnf("telemetry: failed to gracefully shutdown server: %v", err)
		return
	}
	t.Unsubscribe(t.Bus, true)
	close(t.Rx)
	log.Debug("telemetry: completed graceful shutdown of server")
}
