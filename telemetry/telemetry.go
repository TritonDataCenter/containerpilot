package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"utils"

	log "github.com/Sirupsen/logrus"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Sensors.
type Telemetry struct {
	Port          int             `json:"port"`
	Interfaces    json.RawMessage `json:"interfaces,omitempty"` // optional override
	Tags          []string        `json:"tags,omitempty"`
	SensorConfigs json.RawMessage `json:"sensors,omitempty"`
	Sensors       []*Sensor
	IpAddress     string
	ServiceName   string
	Url           string
	TTL           int
	Poll          int
}

func NewTelemetry(raw json.RawMessage) (*Telemetry, error) {
	t := &Telemetry{
		Port:        9090,
		ServiceName: "containerbuddy",
		Url:         "/metrics",
		TTL:         15,
		Poll:        5,
	}
	if err := json.Unmarshal(raw, t); err != nil {
		return nil, errors.New("Telemetry configuration error: %v, err")
	}
	if ipAddress, err := utils.IpFromInterfaces(t.Interfaces); err != nil {
		return nil, err
	} else {
		t.IpAddress = ipAddress
	}
	// note that we don't return an error if there are no sensors
	// because the prometheus handler will still pick up metrics
	// internal to Containerbuddy (i.e. the golang runtime)
	if t.SensorConfigs != nil {
		if sensors, err := NewSensors(t.SensorConfigs); err != nil {
			return nil, err
		} else {
			t.Sensors = sensors
		}
	}
	return t, nil
}

func (t *Telemetry) Serve() {
	http.Handle(t.Url, prometheus.Handler())
	listen := fmt.Sprintf("%s:%v", t.IpAddress, t.Port)
	log.Debugf("Telemetry listening on %v\n", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}
