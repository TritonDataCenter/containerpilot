package telemetry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/commands"
	"github.com/prometheus/client_golang/prometheus"
)

/*
The prometheus client library doesn't expose any of the internals of the
collectors, so we can't ask them directly to find out if we've recorded metrics
in our tests. So for those tests we'll stand up a test HTTP server and give it
the prometheus handler and then check the results of a GET.
*/

func TestSensorPollAction(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()
	cmd, _ := commands.NewCommand("./testdata/test.sh measureStuff", "0")
	sensor := &Sensor{
		Type:     "counter",
		checkCmd: cmd,
		collector: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "telemetry",
			Subsystem: "sensors",
			Name:      "TestSensorPollAction",
			Help:      "help",
		})}
	prometheus.MustRegister(sensor.collector)
	sensor.PollAction()
	resp := getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_TestSensorPollAction 42") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
}

func TestSensorBadPollAction(t *testing.T) {
	cmd, _ := commands.NewCommand("./testdata/doesNotExist.sh", "0")
	sensor := &Sensor{checkCmd: cmd}
	sensor.PollAction() // logs but no crash
}

func TestSensorBadRecord(t *testing.T) {
	cmd, _ := commands.NewCommand("./testdata/test.sh doStuff --debug", "0")
	sensor := &Sensor{checkCmd: cmd}
	sensor.PollAction() // logs but no crash
}

func TestSensorRecordCounter(t *testing.T) {
	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()

	sensor := &Sensor{
		Type: "counter",
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
		Type: "gauge",
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
		Type: "histogram",
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
		Type: "summary",
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

func TestSensorObserve(t *testing.T) {

	cmd1, _ := commands.NewCommand("./testdata/test.sh doStuff --debug", "1s")
	sensor := &Sensor{checkCmd: cmd1}
	if val, err := sensor.observe(); err != nil {
		t.Fatalf("Unexpected error from sensor check: %s", err)
	} else if val != "Running doStuff with args: --debug\n" {
		t.Fatalf("Unexpected output from sensor check: %s", val)
	}

	// Ensure we can run it more than once
	if _, err := sensor.observe(); err != nil {
		t.Fatalf("Unexpected error from sensor check (x2): %s", err)
	}

	// Ensure bad commands return error
	cmd2, _ := commands.NewCommand("./testdata/doesNotExist.sh", "0")
	sensor = &Sensor{checkCmd: cmd2}
	if val, err := sensor.observe(); err == nil {
		t.Fatalf("Expected error from sensor check but got %s", val)
	} else if err.Error() != "fork/exec ./testdata/doesNotExist.sh: no such file or directory" {
		t.Fatalf("Unexpected error from invalid sensor check: %s", err)
	}

}

func TestSensorParse(t *testing.T) {
	jsonFragment := `[{
	"namespace": "telemetry",
	"subsystem": "sensors",
	"name": "%s",
	"help": "help",
	"type": "%s",
	"poll": 10,
	"check": ["/bin/sensor.sh"]
}]`

	test1Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_counter", "counter"))
	collector := parseSensors(t, test1Json)[0].collector
	if _, ok := collector.(prometheus.Counter); !ok {
		t.Fatalf("Incorrect collector; expected Counter but got %v", collector)
	}

	test2Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_gauge", "gauge"))
	collector = parseSensors(t, test2Json)[0].collector
	if _, ok := collector.(prometheus.Gauge); !ok {
		t.Fatalf("Incorrect collector; expected Gauge but got %v", collector)
	}

	test3Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_histogram", "histogram"))
	collector = parseSensors(t, test3Json)[0].collector
	if _, ok := collector.(prometheus.Histogram); !ok {
		t.Fatalf("Incorrect collector; expected Histogram but got %v", collector)
	}

	test4Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_summary", "summary"))
	collector = parseSensors(t, test4Json)[0].collector
	if _, ok := collector.(prometheus.Summary); !ok {
		t.Fatalf("Incorrect collector; expected Summary but got %v", collector)
	}
}

// invalid collector type
func TestSensorBadType(t *testing.T) {
	jsonFragment := []byte(`[{
	"namespace": "telemetry",
	"subsystem": "sensors",
	"name": "TestSensorBadType",
	"type": "nonsense",
	"check": "true"}]`)

	if sensors, err := NewSensors(decodeJSONRawSensor(t, jsonFragment)); err == nil {
		t.Fatalf("Did not get expected error from parsing sensors: %v", sensors)
	}
}

// invalid metric name
func TestSensorBadName(t *testing.T) {
	jsonFragment := []byte(`[{
	"namespace": "telemetry",
	"subsystem": "sensors",
	"name": "Test.Sensor.Bad.Name",
	"type": "counter",
	"check": "true"}]`)

	if sensors, err := NewSensors(decodeJSONRawSensor(t, jsonFragment)); err == nil {
		t.Fatalf("Did not get expected error from parsing sensors: %v", sensors)
	}
}

// partial metric name parses ok and write out as expected
func TestSensorPartialName(t *testing.T) {

	testServer := httptest.NewServer(prometheus.UninstrumentedHandler())
	defer testServer.Close()

	jsonFragment := []byte(`[{
	"name": "telemetry_sensors_partial_name",
	"help": "help text",
	"type": "counter",
	"check": "true"}]`)
	sensor := parseSensors(t, jsonFragment)[0]
	if _, ok := sensor.collector.(prometheus.Counter); !ok {
		t.Fatalf("Incorrect collector; expected Counter but got %v", sensor.collector)
	}

	sensor.record("1")
	resp := getFromTestServer(t, testServer)
	if strings.Count(resp, "telemetry_sensors_partial_name 1") != 1 {
		t.Fatalf("Failed to get match for sensor in response: %s", resp)
	}
}

func decodeJSONRawSensor(t *testing.T, testJSON json.RawMessage) []interface{} {
	var raw []interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("Unexpected error decoding JSON:\n%s\n%v", testJSON, err)
	}
	return raw
}

func parseSensors(t *testing.T, testJSON json.RawMessage) []*Sensor {
	if sensors, err := NewSensors(decodeJSONRawSensor(t, testJSON)); err != nil {
		t.Fatalf("Could not parse sensor JSON: %s", err)
	} else {
		if len(sensors) == 0 {
			t.Fatalf("Did not get a valid sensor from JSON.")
		}
		return sensors
	}
	return nil
}
