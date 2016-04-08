package telemetry

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"utils"

	log "github.com/Sirupsen/logrus"
)

// Telemetry represents the service to advertise for finding the metrics
// endpoint, and the collection of Sensors.
type Telemetry struct {
	Port        int             `json:"port"`
	Interfaces  json.RawMessage `json:"interfaces"` // optional override
	Tags        []string        `json:"tags,omitempty"`
	Sensors     []*Sensor       `json:"sensors"`
	IpAddress   string
	ServiceName string
	Url         string
	TTL         int
	Poll        int
}

func (m *Telemetry) Parse() error {
	if ipAddress, err := utils.IpFromInterfaces(m.Interfaces); err != nil {
		return err
	} else {
		m.IpAddress = ipAddress
	}
	if m.Port == 0 {
		m.Port = 9090
	}

	// set hard-coded service values
	m.ServiceName = "containerbuddy"
	m.Url = "/metrics"
	m.TTL = 15
	m.Poll = 5

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
	log.Debugf("Telemetry listening on %v\n", listen)
	log.Fatal(http.ListenAndServe(listen, nil))
}
