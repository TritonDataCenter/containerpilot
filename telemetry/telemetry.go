package telemetry

import (
	"net"
	"net/http"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Sensors.
type Telemetry struct {
	Sensors   []*Sensor
	Path      string
	heartbeat time.Duration
	mux       *http.ServeMux
	lock      sync.RWMutex
	addr      net.TCPAddr
	listening bool
}

// NewTelemetry configures a new prometheus Telemetry server
func NewTelemetry(cfg *TelemetryConfig) (*Telemetry, error) {
	t := &Telemetry{
		Path:    "/metrics", // TODO hard-coded?
		lock:    sync.RWMutex{},
		Sensors: []*Sensor{},
	}

	t.addr = cfg.addr
	t.mux = http.NewServeMux()
	t.mux.Handle(t.Path, prometheus.Handler())
	for _, sensorCfg := range cfg.SensorConfigs {
		sensor, err := NewSensor(sensorCfg)
		if err != nil {
			return t, err
		}
		t.Sensors = append(t.Sensors, sensor)
	}
	return t, nil
}

var listener net.Listener

// Serve starts serving the telemetry service
func (t *Telemetry) Serve() {
	t.lock.Lock()
	defer t.lock.Unlock()

	// No-op if we've created the server previously.
	// TODO: golang's native implementation of http.Server.Server() cannot
	// support graceful reload. We need to select an alternate implementation
	// but in the meantime we need to back-out the change to reloading
	// ref https://github.com/joyent/containerpilot/pull/165
	if listener != nil {
		return
	}
	ln, err := net.Listen(t.addr.Network(), t.addr.String())
	if err != nil {
		log.Fatalf("Error serving telemetry on %s: %v", t.addr.String(), err)
	}
	listener = ln
	t.listening = true
	go func() {
		log.Infof("telemetry: Listening on %s", t.addr.String())
		log.Fatal(http.Serve(listener, t.mux))
		log.Debugf("telemetry: Stopped listening on %s", t.addr.String())
	}()
}

// Shutdown shuts down the telemetry service
func (t *Telemetry) Shutdown() {
	log.Debug("telemetry: shutdown received but currently a no-op")
}
