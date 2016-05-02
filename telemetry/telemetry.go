package telemetry

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Sensors.
type Telemetry struct {
	Port          int           `mapstructure:"port"`
	Interfaces    []interface{} `mapstructure:"interfaces"` // optional override
	Tags          []string      `mapstructure:"tags"`
	SensorConfigs []interface{} `mapstructure:"sensors"`
	Sensors       []*Sensor
	ServiceName   string
	URL           string
	TTL           int
	Poll          int
	mux           *http.ServeMux
	lock          sync.RWMutex
	listen        net.Listener
	addr          net.TCPAddr
	listening     bool
}

// NewTelemetry configures a new prometheus Telemetry server
func NewTelemetry(raw interface{}) (*Telemetry, error) {
	t := &Telemetry{
		Port:        9090,
		ServiceName: "containerpilot",
		URL:         "/metrics",
		TTL:         15,
		Poll:        5,
		lock:        sync.RWMutex{},
	}

	if err := utils.DecodeRaw(raw, t); err != nil {
		return nil, fmt.Errorf("Telemetry configuration error: %v", err)
	}
	ipAddress, err := utils.IPFromInterfaces(t.Interfaces)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(ipAddress)
	t.addr = net.TCPAddr{IP: ip, Port: t.Port}
	t.mux = http.NewServeMux()
	t.mux.Handle(t.URL, prometheus.Handler())
	// note that we don't return an error if there are no sensors
	// because the prometheus handler will still pick up metrics
	// internal to ContainerPilot (i.e. the golang runtime)
	if t.SensorConfigs != nil {
		sensors, err := NewSensors(t.SensorConfigs)
		if err != nil {
			return nil, err
		}
		t.Sensors = sensors
	}
	return t, nil
}

// Serve starts serving the telemetry service
func (t *Telemetry) Serve() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.listening {
		return
	}
	ln, err := net.Listen(t.addr.Network(), t.addr.String())
	if err != nil {
		log.Errorf("Error serving telemetry on %s: %v", t.addr.String(), err)
	}
	t.listen = ln
	t.listening = true
	go func() {
		log.Debugf("telemetry: Listening on %s\n", t.addr.String())
		log.Fatal(http.Serve(t.listen, t.mux))
		log.Debugf("telemetry: Stopped listening on %s\n", t.addr.String())
	}()
}

// Shutdown shuts down the telemetry service
func (t *Telemetry) Shutdown() {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.listening {
		t.listen.Close()
		t.listening = false
	}
}
