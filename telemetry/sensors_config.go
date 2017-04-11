package telemetry

import (
	"fmt"
	"time"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// A SensorConfig is a single measurement of the application.
type SensorConfig struct {
	Namespace string      `mapstructure:"namespace"`
	Subsystem string      `mapstructure:"subsystem"`
	Name      string      `mapstructure:"name"`
	Help      string      `mapstructure:"help"` // help string returned by API
	Type      string      `mapstructure:"type"`
	Poll      int         `mapstructure:"interval"` // time in seconds
	Exec      interface{} `mapstructure:"exec"`
	Timeout   string      `mapstructure:"timeout"`

	sensorType SensorType
	poll       time.Duration
	timeout    time.Duration
	exec       *commands.Command
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

	if cfg.Timeout == "" {
		cfg.Timeout = fmt.Sprintf("%ds", cfg.Poll)
	}
	if cfg.Poll <= 0 {
		return fmt.Errorf("sensor[%s].interval must be > 0", cfg.Name)
	}
	poll, err := utils.ParseDuration(cfg.Poll)
	if err != nil {
		return fmt.Errorf("unable to parse sensor[%s].interval: %v", cfg.Name, err)
	}
	cfg.poll = poll

	timeout, err := utils.GetTimeout(cfg.Timeout)
	if err != nil {
		return fmt.Errorf("unable to parse sensor[%s].timeout: %v", cfg.Name, err)
	}
	cfg.timeout = timeout
	check, err := commands.NewCommand(cfg.Exec, cfg.timeout, nil)
	if err != nil {
		return fmt.Errorf("unable to create sensor[%s].exec: %v", cfg.Name, err)
	}
	check.Name = fmt.Sprintf("%s.sensor", cfg.Name)
	cfg.exec = check

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
