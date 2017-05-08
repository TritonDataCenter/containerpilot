package telemetry

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests/assert"
	"github.com/joyent/containerpilot/tests/mocks"
	"github.com/prometheus/client_golang/prometheus"
)

/*
The prometheus client library doesn't expose any of the internals of the
collectors, so we can't ask them directly to find out if we've recorded metrics
in our tests. So for those tests we'll stand up a test HTTP server and give it
the prometheus handler and then check the results of a GET.
*/

func TestSensorRun(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	cfg := &SensorConfig{
		Namespace: "telemetry",
		Subsystem: "sensors",
		Name:      "TestSensorObserve",
		Help:      "help",
		Type:      "counter",
		Poll:      1,
		Exec:      "true",
	}
	cfg.Validate()
	sensor := NewSensor(cfg)

	bus := events.NewEventBus()
	ds := mocks.NewDebugSubscriber(bus, 6)
	ds.Run(0)
	sensor.Run(bus)

	exitOk := events.Event{events.ExitSuccess, fmt.Sprintf("%s.sensor", sensor.Name)}
	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-sensor-poll", sensor.Name)}
	record := events.Event{events.Metric, fmt.Sprintf("%s|84", sensor.Name)}

	bus.Publish(poll)
	bus.Publish(poll) // Ensure we can run it more than once
	bus.Publish(record)
	sensor.Close()
	ds.Close()

	got := map[events.Event]int{}
	for _, result := range ds.Results {
		got[result]++
	}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}

	resp := getFromTestServer(t, testServer)
	assert.Equal(t,
		strings.Count(resp, "telemetry_sensors_TestSensorObserve 84"), 1,
		"failed to get match for sensor in response")
}

// TestSensorProcessMetric covers the same ground as the 4 collector-
// specific tests below, but checks the unhappy path.
func TestSensorProcessMetric(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	cfg := &SensorConfig{
		Namespace: "telemetry",
		Subsystem: "sensors",
		Name:      "TestSensorProcessMetric",
		Help:      "help",
		Type:      "gauge",
		Poll:      1,
		Exec:      "true",
	}
	cfg.Validate()
	sensor := NewSensor(cfg)
	testFunc := func(input, expected string) bool {
		sensor.processMetric(input)
		resp := getFromTestServer(t, testServer)
		return strings.Count(resp, expected) == 1
	}

	t.Run("record Ok", func(t *testing.T) {
		assert.True(t, testFunc(
			"telemetry_sensors_TestSensorProcessMetric|30.0",
			"telemetry_sensors_TestSensorProcessMetric 30",
		), "failed to get match for sensor in response")
	})
	t.Run("record wrong name", func(t *testing.T) {
		assert.True(t, testFunc(
			"TestSensorProcessMetric|20.0",
			"telemetry_sensors_TestSensorProcessMetric 30",
		), "should not have updated sensor value")
	})
	t.Run("invalid record", func(t *testing.T) {
		assert.True(t, testFunc(
			"telemetry_sensors_TestSensorProcessMetric",
			"telemetry_sensors_TestSensorProcessMetric 30",
		), "should not have updated sensor value")
	})
	t.Run("non-numeric record", func(t *testing.T) {
		assert.True(t, testFunc(
			"telemetry_sensors_TestSensorProcessMetric|xxx",
			"telemetry_sensors_TestSensorProcessMetric 30",
		), "should not have updated sensor value")
	})

}

func TestSensorRecordCounter(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	sensor := &Sensor{
		Type: Counter,
		collector: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "telemetry",
			Subsystem: "sensors",
			Name:      "TestSensorRecordCounter",
			Help:      "help",
		})}
	prometheus.MustRegister(sensor.collector)
	testFunc := func(input, expected string) bool {
		sensor.record(input)
		resp := getFromTestServer(t, testServer)
		return strings.Count(resp, expected) == 1
	}
	t.Run("record ok", func(t *testing.T) {
		assert.True(t, testFunc(
			"1", "telemetry_sensors_TestSensorRecordCounter 1"),
			"failed to update sensor")
	})
	t.Run("record update", func(t *testing.T) {
		assert.True(t, testFunc(
			"2", "telemetry_sensors_TestSensorRecordCounter 3"),
			"failed to update sensor")
	})
}

func TestSensorRecordGauge(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	sensor := &Sensor{
		Type: Gauge,
		collector: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "telemetry",
			Subsystem: "sensors",
			Name:      "TestSensorRecordGauge",
			Help:      "help",
		})}
	prometheus.MustRegister(sensor.collector)

	testFunc := func(input, expected string) bool {
		sensor.record(input)
		resp := getFromTestServer(t, testServer)
		return strings.Count(resp, expected) == 1
	}
	t.Run("record ok", func(t *testing.T) {
		assert.True(t, testFunc(
			"1.2", "telemetry_sensors_TestSensorRecordGauge 1.2"),
			"failed to update sensor")
	})
	t.Run("record update", func(t *testing.T) {
		assert.True(t, testFunc(
			"2.3", "telemetry_sensors_TestSensorRecordGauge 2.3"),
			"failed to update sensor")
	})
}

func TestSensorRecordHistogram(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()

	sensor := &Sensor{
		Type: Histogram,
		collector: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: "telemetry",
			Subsystem: "sensors",
			Name:      "TestSensorRecordHistogram",
			Help:      "help",
		})}
	prometheus.MustRegister(sensor.collector)
	patt := `telemetry_sensors_TestSensorRecordHistogram_bucket{le="([\.0-9|\+Inf]*)"} ([1-9])`

	testFunc := func(input string, expected [][]string) bool {
		sensor.record(input)
		resp := getFromTestServer(t, testServer)
		return checkBuckets(resp, patt, expected)
	}
	t.Run("record ok", func(t *testing.T) {
		assert.True(t, testFunc("1.2",
			[][]string{{"2.5", "1"}, {"5", "1"}, {"10", "1"}, {"+Inf", "1"}}),
			"failed to update sensor")
	})
	t.Run("record add", func(t *testing.T) {
		assert.True(t, testFunc("1.2",
			[][]string{{"2.5", "2"}, {"5", "2"}, {"10", "2"}, {"+Inf", "2"}}),
			"failed to update sensor")
	})
	t.Run("record overlap", func(t *testing.T) {
		assert.True(t, testFunc("4.5",
			[][]string{{"2.5", "2"}, {"5", "3"}, {"10", "3"}, {"+Inf", "3"}}),
			"failed to update sensor")
	})
}

func TestSensorRecordSummary(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	sensor := &Sensor{
		Type: Summary,
		collector: prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace: "telemetry",
			Subsystem: "sensors",
			Name:      "TestSensorRecordSummary",
			Help:      "help",
		})}
	prometheus.MustRegister(sensor.collector)
	patt := `telemetry_sensors_TestSensorRecordSummary{quantile="([\.0-9]*)"} ([0-9\.]*)`

	t.Run("record ok", func(t *testing.T) {
		// need a bunch of metrics to make quantiles make any sense
		for i := 1; i <= 10; i++ {
			sensor.record(fmt.Sprintf("%v", i))
		}
		resp := getFromTestServer(t, testServer)
		expected := [][]string{{"0.5", "5"}, {"0.9", "9"}, {"0.99", "10"}}
		assert.True(t, checkBuckets(resp, patt, expected),
			"failed to get match for sensor in response")
	})
	t.Run("record update", func(t *testing.T) {
		for i := 1; i <= 5; i++ {
			// add a new record for each one in the bottom half
			sensor.record(fmt.Sprintf("%v", i))
		}
		resp := getFromTestServer(t, testServer)
		expected := [][]string{{"0.5", "4"}, {"0.9", "9"}, {"0.99", "10"}}
		assert.True(t, checkBuckets(resp, patt, expected),
			"failed to get match for sensor in response")
	})
}

// test helpers

func checkBuckets(resp, patt string, expected [][]string) bool {
	re := regexp.MustCompile(patt)
	matches := re.FindAllStringSubmatch(resp, -1)
	var buckets [][]string
	if len(matches) != len(expected) {
		fmt.Printf("%v matches vs %v expected\n", len(matches), len(expected))
		return false
	}
	for _, m := range matches {
		buckets = append(buckets, []string{m[1], m[2]})
	}
	var ok bool
	if ok = reflect.DeepEqual(buckets, expected); !ok {
		fmt.Println("Match content:")
		for _, m := range matches {
			fmt.Println(m)
		}
		fmt.Printf("Expected:\n%v\n-----Actual:\n%v\n", expected, buckets)
	}
	return ok
}

func getFromTestServer(t *testing.T, testServer *httptest.Server) string {
	if res, err := http.Get(testServer.URL); err != nil {
		t.Fatal(err)
	} else {
		defer res.Body.Close()
		if resp, err := ioutil.ReadAll(res.Body); err != nil {
			t.Fatal(err)
		} else {
			response := string(resp)
			return response
		}
	}
	return ""
}
