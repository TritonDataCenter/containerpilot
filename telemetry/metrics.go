package telemetry

import (
	"context"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
	"github.com/prometheus/client_golang/prometheus"
)

const eventBufferSize = 1000

// go:generate stringer -type SensorType

// SensorType is an enum for Prometheus sensor types
type SensorType int

// SensorType enum
const (
	Counter SensorType = iota
	Gauge
	Histogram
	Summary
)

// Sensor manages state of periodic sensors.
type Sensor struct {
	Name      string
	Type      SensorType
	collector prometheus.Collector

	events.EventHandler // Event handling
}

// NewSensor creates a Sensor from a validated SensorConfig
func NewSensor(cfg *SensorConfig) *Sensor {
	sensor := &Sensor{
		Name:      cfg.fullName,
		Type:      cfg.sensorType,
		collector: cfg.collector,
	}
	sensor.Rx = make(chan events.Event, eventBufferSize)
	return sensor
}

func (sensor *Sensor) processMetric(event string) {
	metric := strings.Split(event, "|")
	if len(metric) < 2 {
		log.Errorf("sensor: invalid metric format: %v", event)
		return
	}
	metricKey := metric[0]
	metricVal := metric[1]
	if sensor.Name == metricKey {
		sensor.record(metricVal)
	}
}

func (sensor *Sensor) record(metricValue string) {
	if val, err := strconv.ParseFloat(
		strings.TrimSpace(metricValue), 64); err != nil {
		log.Errorf("sensor produced non-numeric value: %v: %v", metricValue, err)
	} else {
		// we should use a type switch here but the prometheus collector
		// implementations are themselves interfaces and not structs,
		// so that doesn't work.
		switch sensor.Type {
		case Counter:
			sensor.collector.(prometheus.Counter).Add(val)
		case Gauge:
			sensor.collector.(prometheus.Gauge).Set(val)
		case Histogram:
			sensor.collector.(prometheus.Histogram).Observe(val)
		case Summary:
			sensor.collector.(prometheus.Summary).Observe(val)
		}
	}
}

// Run executes the event loop for the Sensor
func (sensor *Sensor) Run(bus *events.EventBus) {
	sensor.Subscribe(bus)
	sensor.Bus = bus
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() {
			cancel()
			sensor.Unsubscribe(sensor.Bus)
		}()
		for {
			select {
			case event, ok := <-sensor.Rx:
				if !ok {
					return
				}
				switch event.Code {
				case events.Metric:
					sensor.processMetric(event.Source)
				default:
					switch event {
					case
						events.Event{events.Quit, sensor.Name},
						events.QuitByClose,
						events.GlobalShutdown:
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
