package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"utils"

	log "github.com/Sirupsen/logrus"
)

// A Sensor is a single measurement of the application.
type Sensor struct {
	Namespace string          `json:"namespace"`
	Subsystem string          `json:"subsystem"`
	Name      string          `json:"name"`
	Help      string          `json:"help"` // help string returned by API
	Type      string          `json:"type"`
	Poll      int             `json:"poll"` // time in seconds
	CheckExec json.RawMessage `json:"check"`
	checkCmd  *exec.Cmd
	collector prometheus.Collector
}

// PollTime implements Pollable for Sensor
// It returns the sensor's poll interval.
func (s Sensor) PollTime() int {
	return s.Poll
}

// PollAction implements Pollable for Sensor.
func (s *Sensor) PollAction() {
	if metricValue, err := s.observe(); err == nil {
		s.record(metricValue)
	} else {
		log.Errorln(err)
	}
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

func NewSensors(raw json.RawMessage) ([]*Sensor, error) {
	sensors := make([]*Sensor, 0)
	if err := json.Unmarshal(raw, &sensors); err != nil {
		return nil, errors.New("Sensor configuration error: %v, err")
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
		if err := prometheus.Register(s.collector); err != nil {
			return nil, err
		}
	}
	return sensors, nil
}
