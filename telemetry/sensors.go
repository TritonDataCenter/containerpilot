package telemetry

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
	"github.com/prometheus/client_golang/prometheus"
)

// A Sensor is a single measurement of the application.
type Sensor struct {
	Namespace string      `mapstructure:"namespace"`
	Subsystem string      `mapstructure:"subsystem"`
	Name      string      `mapstructure:"name"`
	Help      string      `mapstructure:"help"` // help string returned by API
	Type      string      `mapstructure:"type"`
	Poll      int         `mapstructure:"poll"` // time in seconds
	CheckExec interface{} `mapstructure:"check"`
	checkCmd  *exec.Cmd
	collector prometheus.Collector
}

// PollTime implements Pollable for Sensor
// It returns the sensor's poll interval.
func (s Sensor) PollTime() time.Duration {
	return time.Duration(s.Poll) * time.Second
}

// PollAction implements Pollable for Sensor.
func (s *Sensor) PollAction() {
	if metricValue, err := s.observe(); err == nil {
		s.record(metricValue)
	} else {
		log.Errorln(err)
	}
}

// PollStop does nothing in a Sensor
func (s *Sensor) PollStop() {
	// Nothing to do
}

func (s *Sensor) observe() (string, error) {
	defer func() {
		// reset command object because it can't be reused
		s.checkCmd = utils.ArgsToCmd(s.checkCmd.Args)
	}()

	// we'll pass stderr to the container's stderr, but stdout must
	// be "clean" and not have anything other than what we intend
	// to write to our collector.
	s.checkCmd.Stderr = os.Stderr
	if out, err := s.checkCmd.Output(); err != nil {
		return "", err
	} else {
		return string(out[:]), nil
	}
}

func (s Sensor) record(metricValue string) {
	if val, err := strconv.ParseFloat(
		strings.TrimSpace(metricValue), 64); err != nil {
		log.Errorln(err)
	} else {
		// we should use a type switch here but the prometheus collector
		// implementations are themselves interfaces and not structs,
		// so that doesn't work...
		switch {
		case s.Type == "counter":
			s.collector.(prometheus.Counter).Add(val)
		case s.Type == "gauge":
			s.collector.(prometheus.Gauge).Set(val)
		case s.Type == "histogram":
			s.collector.(prometheus.Histogram).Observe(val)
		case s.Type == "summary":
			s.collector.(prometheus.Summary).Observe(val)
		default:
			// ...which is why we end up logging the fall-thru
			log.Errorf("Invalid sensor type: %s\n", s.Type)
		}
	}
}

func NewSensors(raw []interface{}) ([]*Sensor, error) {
	sensors := make([]*Sensor, 0)
	if err := utils.DecodeRaw(raw, &sensors); err != nil {
		return nil, fmt.Errorf("Sensor configuration error: %v", err)
	}
	for _, s := range sensors {
		if check, err := utils.ParseCommandArgs(s.CheckExec); err != nil {
			return nil, err
		} else {
			s.checkCmd = check
		}
		// the prometheus client lib's API here is baffling... they don't expose
		// an interface or embed their Opts type in each of the Opts "subtypes",
		// so we can't share the initialization.
		switch {
		case s.Type == "counter":
			s.collector = prometheus.NewCounter(prometheus.CounterOpts{
				Namespace: s.Namespace,
				Subsystem: s.Subsystem,
				Name:      s.Name,
				Help:      s.Help,
			})
		case s.Type == "gauge":
			s.collector = prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: s.Namespace,
				Subsystem: s.Subsystem,
				Name:      s.Name,
				Help:      s.Help,
			})
		case s.Type == "histogram":
			s.collector = prometheus.NewHistogram(prometheus.HistogramOpts{
				Namespace: s.Namespace,
				Subsystem: s.Subsystem,
				Name:      s.Name,
				Help:      s.Help,
			})
		case s.Type == "summary":
			s.collector = prometheus.NewSummary(prometheus.SummaryOpts{
				Namespace: s.Namespace,
				Subsystem: s.Subsystem,
				Name:      s.Name,
				Help:      s.Help,
			})
		default:
			return nil, fmt.Errorf("Invalid sensor type: %s\n", s.Type)
		}
		// we're going to unregister before every attempt to register
		// so that we can reload config
		prometheus.Unregister(s.collector)
		if err := prometheus.Register(s.collector); err != nil {
			return nil, err
		}
	}
	return sensors, nil
}
