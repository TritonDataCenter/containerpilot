package metrics

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"os"
	"os/exec"
	"strconv"
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
func (s Sensor) PollAction() {
	if metricValue, err := s.getMetrics(); err != nil {
		s.record(metricValue)
	} else {
		log.Errorln(err)
	}
}

func (s *Sensor) getMetrics() (string, error) {

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
	if val, err := strconv.ParseFloat(metricValue, 64); err != nil {
		log.Errorln(err)
	} else {
		switch collector := s.collector.(type) {
		case prometheus.Counter:
			collector.Add(val)
		case prometheus.Gauge:
			collector.Set(val)
		case prometheus.Histogram:
			collector.Observe(val)
		case prometheus.Summary:
			collector.Observe(val)
		}
	}
}

func (s *Sensor) Parse() (err error) {
	if check, err := utils.ParseCommandArgs(s.CheckExec); err != nil {
		return err
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
		return fmt.Errorf("Invalid sensor type: %s\n", s.Type)
	}

	// MustRegister panics rather than returning an error (thanks!) if we
	// register an invalid collector, even just an invalid name. This would give
	// end-users a big ugly golang stack trace that they have to dig the error
	// message out of. So we'll catch the panic so we can return an error.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	prometheus.MustRegister(s.collector)
	return
}
