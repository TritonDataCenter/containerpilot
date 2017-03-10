package telemetry

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/events"
	"github.com/prometheus/client_golang/prometheus"
)

const eventBufferSize = 1000

// go:generate stringer -type SensorType
type SensorType int

const (
	Counter SensorType = iota
	Gauge
	Histogram
	Summary
)

type Sensor struct {
	Name      string
	Type      SensorType
	exec      *commands.Command
	poll      time.Duration
	collector prometheus.Collector

	events.EventHandler // Event handling
}

// NewSensor ...
func NewSensor(cfg *SensorConfig) (*Sensor, error) {
	sensor := &Sensor{
		Name:      cfg.Name,
		Type:      cfg.sensorType,
		exec:      cfg.exec,
		poll:      cfg.poll,
		collector: cfg.collector,
	}
	sensor.Rx = make(chan events.Event, eventBufferSize)
	sensor.Flush = make(chan bool)
	return sensor, nil
}

// SensorHealth runs the health sensor
func (sensor *Sensor) Observe(ctx context.Context) {
	// TODO: this should be replaced with the async Run once
	// the control plane is available for Sensors to POST to
	output := sensor.exec.RunAndWaitForOutput(ctx, sensor.Bus)
	sensor.record(output)
}

func (sensor *Sensor) record(metricValue string) {
	if val, err := strconv.ParseFloat(
		strings.TrimSpace(metricValue), 64); err != nil {
		log.Errorf("sensor produced non-numeric value: %v", metricValue)
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

	pollSource := fmt.Sprintf("%s-sensor-poll", sensor.Name)
	events.NewEventTimer(ctx, sensor.Rx, sensor.poll, pollSource)

	go func() {
		for {
			event := <-sensor.Rx
			log.Debug(event)
			switch event {
			case events.Event{events.TimerExpired, pollSource}:
				sensor.Observe(ctx)
			case
				events.Event{events.Quit, sensor.Name},
				events.QuitByClose,
				events.GlobalShutdown:
				sensor.Unsubscribe(sensor.Bus)
				close(sensor.Rx)
				cancel()
				sensor.Flush <- true
				return
			}
		}
	}()
}
