package telemetry

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"utils"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Sensors.
type Telemetry struct {
	ServiceName string          `json:"name"`
	Url         string          `json:"url"`
	Port        int             `json:"port"`
	TTL         int             `json:"ttl"`
	Poll        int             `json:"poll"`
	Interfaces  json.RawMessage `json:"interfaces"` // optional override
	Tags        []string        `json:"tags,omitempty"`
	Sensors     []*Sensor       `json:"sensors"`
	IpAddress   string
}

func (m *Telemetry) Parse() error {
	if ipAddress, err := utils.IpFromInterfaces(m.Interfaces); err != nil {
		return err
	} else {
		m.IpAddress = ipAddress
	}
	// note that we don't return an error if there are no sensors
	// because the prometheus handler will still pick up metrics
	// internal to Containerbuddy (i.e. the golang runtime)
	for _, sensor := range m.Sensors {
		if err := sensor.Parse(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Telemetry) Serve() {
	http.Handle(m.Url, prometheus.Handler())
	listen := fmt.Sprintf("%s:%v", m.IpAddress, m.Port)
	http.ListenAndServe(listen, nil)
}
