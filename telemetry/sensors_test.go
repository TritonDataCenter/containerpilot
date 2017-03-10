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

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
	"github.com/prometheus/client_golang/prometheus"
)

/*
The prometheus client library doesn't expose any of the internals of the
collectors, so we can't ask them directly to find out if we've recorded metrics
in our tests. So for those tests we'll stand up a test HTTP server and give it
the prometheus handler and then check the results of a GET.
*/

func TestSensorObserve(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	cfg := &SensorConfig{
		Namespace: "telemetry",
		Subsystem: "sensors",
		Name:      "TestSensorObserve",
		Help:      "help",
		Type:      "counter",
		Poll:      1,
		Exec:      "./testdata/test.sh measureStuff",
		Timeout:   "100ms",
	}

	got := runSensorTest(cfg)
	exitOk := events.Event{events.ExitSuccess, fmt.Sprintf("%s.sensor", cfg.Name)}
	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-sensor-poll", cfg.Name)}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}

	resp := getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorObserve 84") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
}

func TestSensorBadExec(t *testing.T) {
	cfg := &SensorConfig{
		Namespace: "telemetry",
		Subsystem: "sensors",
		Name:      "TestSensorBadExec",
		Help:      "help",
		Type:      "counter",
		Poll:      1,
		Exec:      "./testdata/doesNotExist.sh",
		Timeout:   "100ms",
	}
	got := runSensorTest(cfg)
	exitFail := events.Event{events.ExitFailed, fmt.Sprintf("%s.sensor", cfg.Name)}
	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-sensor-poll", cfg.Name)}
	if got[exitFail] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 failed poll events but got %v", got)
	}
}

func TestSensorBadRecord(t *testing.T) {
	log.SetLevel(log.WarnLevel) // suppress test noise
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	cfg := &SensorConfig{
		Namespace: "telemetry",
		Subsystem: "sensors",
		Name:      "TestSensorBadRecord",
		Help:      "help",
		Type:      "counter",
		Poll:      1,
		Exec:      "./testdata/test.sh doStuff --debug",
		Timeout:   "100ms",
	}
	got := runSensorTest(cfg)
	exitOk := events.Event{events.ExitSuccess, fmt.Sprintf("%s.sensor", cfg.Name)}
	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-sensor-poll", cfg.Name)}
	if got[exitOk] != 2 || got[poll] != 2 || got[events.QuitByClose] != 1 {
		t.Fatalf("expected 2 successful poll events but got %v", got)
	}
	resp := getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorBadRecord 0") != 1 {
		t.Fatalf("expected 0-value sensor data for TestSensorBadRecord but got: %s", resp)
	}
}

func runSensorTest(cfg *SensorConfig) map[events.Event]int {
	bus := events.NewEventBus()
	ds := events.NewDebugSubscriber(bus, 5)
	ds.Run(0)
	cfg.Validate()
	sensor, _ := NewSensor(cfg)
	sensor.Run(bus)

	poll := events.Event{events.TimerExpired, fmt.Sprintf("%s-sensor-poll", cfg.Name)}
	bus.Publish(poll)
	bus.Publish(poll) // Ensure we can run it more than once
	sensor.Close()
	ds.Close()

	got := map[events.Event]int{}
	for _, result := range ds.Results {
		got[result]++
	}
	return got
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
	sensor.record("1")
	resp := getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorRecordCounter 1") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
	sensor.record("2")
	resp = getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorRecordCounter 3") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
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
	sensor.record("1.2")
	resp := getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorRecordGauge 1.2") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
	sensor.record("2.3")
	resp = getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorRecordGauge 2.3") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
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

	sensor.record("1.2")
	resp := getFromTestServer(t, testServer)
	expected := [][]string{{"2.5", "1"}, {"5", "1"}, {"10", "1"}, {"+Inf", "1"}}
	if !checkBuckets(resp, patt, expected) {
		t.Fatalf("Failed to get match for sensor in response")
	}
	sensor.record("1.2") // same value should add
	resp = getFromTestServer(t, testServer)
	expected = [][]string{{"2.5", "2"}, {"5", "2"}, {"10", "2"}, {"+Inf", "2"}}
	if !checkBuckets(resp, patt, expected) {
		t.Fatalf("Failed to get match for sensor in response")
	}
	sensor.record("4.5") // overlapping should overlap
	resp = getFromTestServer(t, testServer)
	expected = [][]string{{"2.5", "2"}, {"5", "3"}, {"10", "3"}, {"+Inf", "3"}}
	if !checkBuckets(resp, patt, expected) {
		t.Fatalf("Failed to get match for sensor in response")
	}
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

	// need a bunch of metrics to make quantiles make any sense
	for i := 1; i <= 10; i++ {
		sensor.record(fmt.Sprintf("%v", i))
	}
	resp := getFromTestServer(t, testServer)
	expected := [][]string{{"0.5", "5"}, {"0.9", "9"}, {"0.99", "10"}}
	if !checkBuckets(resp, patt, expected) {
		t.Fatalf("Failed to get match for sensor in response")
	}

	for i := 1; i <= 5; i++ {
		// add a new record for each one in the bottom half
		sensor.record(fmt.Sprintf("%v", i))
	}
	resp = getFromTestServer(t, testServer)
	expected = [][]string{{"0.5", "4"}, {"0.9", "9"}, {"0.99", "10"}}
	if !checkBuckets(resp, patt, expected) {
		t.Fatalf("Failed to get match for sensor in response")
	}
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
