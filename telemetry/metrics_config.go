package telemetry

import (
	"fmt"
	"strings"

	"github.com/tritondatacenter/containerpilot/config/decode"
	"github.com/prometheus/client_golang/prometheus"
)

// A MetricConfig is a single measurement of the application.
type MetricConfig struct {
	Namespace string `mapstructure:"namespace"`
	Subsystem string `mapstructure:"subsystem"`
	Name      string `mapstructure:"name"`
	Help      string `mapstructure:"help"` // help string returned by API
	Type      string `mapstructure:"type"`

	fullName   string // combined name
	metricType MetricType
	collector  prometheus.Collector
}

// NewMetricConfigs creates new metrics from a raw config
func NewMetricConfigs(raw []interface{}) ([]*MetricConfig, error) {
	var metrics []*MetricConfig
	if err := decode.ToStruct(raw, &metrics); err != nil {
		return nil, fmt.Errorf("MetricConfig configuration error: %v", err)
	}
	for _, metric := range metrics {
		if err := metric.Validate(); err != nil {
			return metrics, err
		}
	}
	return metrics, nil
}

// Validate ensures Metric meets all requirements
func (cfg *MetricConfig) Validate() error {

	cfg.fullName = strings.Join([]string{cfg.Namespace, cfg.Subsystem, cfg.Name}, "_")

	// the prometheus client lib's API here is baffling... they don't expose
	// an interface or embed their Opts type in each of the Opts "subtypes",
	// so we can't share the initialization.
	switch cfg.Type {
	case "counter":
		cfg.metricType = Counter
		cfg.collector = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	case "gauge":
		cfg.metricType = Gauge
		cfg.collector = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	case "histogram":
		cfg.metricType = Histogram
		cfg.collector = prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	case "summary":
		cfg.metricType = Summary
		cfg.collector = prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	default:
		return fmt.Errorf("invalid metric type: %s", cfg.Type)
	}
	// we're going to unregister before every attempt to register
	// so that we can reload config
	prometheus.Unregister(cfg.collector)
	return prometheus.Register(cfg.collector)
}
