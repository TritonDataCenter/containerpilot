package telemetry

import (
	"fmt"
	"strings"

	"github.com/joyent/containerpilot/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// A SensorConfig is a single measurement of the application.
type SensorConfig struct {
	Namespace string `mapstructure:"namespace"`
	Subsystem string `mapstructure:"subsystem"`
	Name      string `mapstructure:"name"`
	Help      string `mapstructure:"help"` // help string returned by API
	Type      string `mapstructure:"type"`

	fullName   string // combined name
	sensorType SensorType
	collector  prometheus.Collector
}

// NewSensorConfigs creates new sensors from a raw config
func NewSensorConfigs(raw []interface{}) ([]*SensorConfig, error) {
	var sensors []*SensorConfig
	if err := utils.DecodeRaw(raw, &sensors); err != nil {
		return nil, fmt.Errorf("SensorConfig configuration error: %v", err)
	}
	for _, sensor := range sensors {
		if err := sensor.Validate(); err != nil {
			return sensors, err
		}
	}
	return sensors, nil
}

// Validate ensures Sensor meets all requirements
func (cfg *SensorConfig) Validate() error {

	cfg.fullName = strings.Join([]string{cfg.Namespace, cfg.Subsystem, cfg.Name}, "_")

	// the prometheus client lib's API here is baffling... they don't expose
	// an interface or embed their Opts type in each of the Opts "subtypes",
	// so we can't share the initialization.
	switch cfg.Type {
	case "counter":
		cfg.sensorType = Counter
		cfg.collector = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	case "gauge":
		cfg.sensorType = Gauge
		cfg.collector = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	case "histogram":
		cfg.sensorType = Histogram
		cfg.collector = prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	case "summary":
		cfg.sensorType = Summary
		cfg.collector = prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      cfg.Name,
			Help:      cfg.Help,
		})
	default:
		return fmt.Errorf("invalid sensor type: %s", cfg.Type)
	}
	// we're going to unregister before every attempt to register
	// so that we can reload config
	prometheus.Unregister(cfg.collector)
	if err := prometheus.Register(cfg.collector); err != nil {
		return err
	}

	return nil
}
