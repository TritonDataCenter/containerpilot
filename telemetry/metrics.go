package telemetry

import (
	"context"
	"strconv"
	"strings"

	"github.com/tritondatacenter/containerpilot/events"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const eventBufferSize = 1000

// go:generate stringer -type MetricType

// MetricType is an enum for Prometheus metric types
type MetricType int

// MetricType enum
const (
	Counter MetricType = iota
	Gauge
	Histogram
	Summary
)

// Metric manages state of periodic metrics.
type Metric struct {
	Name      string
	Type      MetricType
	collector prometheus.Collector

	events.Subscriber
}

// NewMetric creates a Metric from a validated MetricConfig
func NewMetric(cfg *MetricConfig) *Metric {
	metric := &Metric{
		Name:      cfg.fullName,
		Type:      cfg.metricType,
		collector: cfg.collector,
	}
	metric.Rx = make(chan events.Event, eventBufferSize)
	return metric
}

func (metric *Metric) processMetric(event string) {
	measurement := strings.Split(event, "|")
	if len(measurement) < 2 {
		log.Errorf("metric: invalid metric format: %v", event)
		return
	}
	metricKey := measurement[0]
	metricVal := measurement[1]
	if metric.Name == metricKey {
		metric.record(metricVal)
	}
}

func (metric *Metric) record(metricValue string) {
	if val, err := strconv.ParseFloat(
		strings.TrimSpace(metricValue), 64); err != nil {
		log.Errorf("metric produced non-numeric value: %v: %v", metricValue, err)
	} else {
		// we should use a type switch here but the prometheus collector
		// implementations are themselves interfaces and not structs,
		// so that doesn't work.
		switch metric.Type {
		case Counter:
			metric.collector.(prometheus.Counter).Add(val)
		case Gauge:
			metric.collector.(prometheus.Gauge).Set(val)
		case Histogram:
			metric.collector.(prometheus.Histogram).Observe(val)
		case Summary:
			metric.collector.(prometheus.Summary).Observe(val)
		}
	}
}

// Run executes the event loop for the Metric
func (metric *Metric) Run(pctx context.Context, bus *events.EventBus) {
	metric.Subscribe(bus)
	ctx, cancel := context.WithCancel(pctx)
	go func() {
		defer func() {
			cancel()
			metric.Unsubscribe()
			metric.Wait()
		}()
		for {
			select {
			case event, ok := <-metric.Rx:
				if !ok {
					return
				}
				switch event.Code {
				case events.Metric:
					metric.processMetric(event.Source)
				default:
					switch event {
					case events.GlobalShutdown, events.QuitByTest:
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
