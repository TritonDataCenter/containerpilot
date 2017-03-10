package telemetry

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestSensorConfigParse(t *testing.T) {
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
		t.Fatalf("incorrect collector; expected Counter but got %v", collector)
	}

	test2Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_gauge", "gauge"))
	collector = parseSensors(t, test2Json)[0].collector
	if _, ok := collector.(prometheus.Gauge); !ok {
		t.Fatalf("incorrect collector; expected Gauge but got %v", collector)
	}

	test3Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_histogram", "histogram"))
	collector = parseSensors(t, test3Json)[0].collector
	if _, ok := collector.(prometheus.Histogram); !ok {
		t.Fatalf("incorrect collector; expected Histogram but got %v", collector)
	}

	test4Json := []byte(fmt.Sprintf(jsonFragment,
		"TestSensorParse_summary", "summary"))
	collector = parseSensors(t, test4Json)[0].collector
	if _, ok := collector.(prometheus.Summary); !ok {
		t.Fatalf("incorrect collector; expected Summary but got %v", collector)
	}
}

// invalid collector type
func TestSensorConfigBadType(t *testing.T) {
	jsonFragment := []byte(`[{
	"namespace": "telemetry",
	"subsystem": "sensors",
	"name": "TestSensorBadType",
	"type": "nonsense",
	"check": "true"}]`)

	if sensors, err := NewSensorConfigs(decodeJSONRawSensor(t, jsonFragment)); err == nil {
		t.Fatalf("did not get expected error from parsing sensors: %v", sensors)
	}
}

// invalid metric name
func TestSensorConfigBadName(t *testing.T) {
	jsonFragment := []byte(`[{
	"namespace": "telemetry",
	"subsystem": "sensors",
	"name": "Test.Sensor.Bad.Name",
	"type": "counter",
	"check": "true"}]`)

	if sensors, err := NewSensorConfigs(decodeJSONRawSensor(t, jsonFragment)); err == nil {
		t.Fatalf("did not get expected error from parsing sensors: %v", sensors)
	}
}

// partial metric name parses ok and write out as expected
func TestSensorConfigPartialName(t *testing.T) {
	jsonFragment := []byte(`[{
	"name": "telemetry_sensors_partial_name",
	"help": "help text",
	"type": "counter",
	"check": "true"}]`)
	sensor := parseSensors(t, jsonFragment)[0]
	if _, ok := sensor.collector.(prometheus.Counter); !ok {
		t.Fatalf("incorrect collector; expected Counter but got %v", sensor.collector)
	}
}

func decodeJSONRawSensor(t *testing.T, testJSON json.RawMessage) []interface{} {
	var raw []interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("unexpected error decoding JSON:\n%s\n%v", testJSON, err)
	}
	return raw
}

func parseSensors(t *testing.T, testJSON json.RawMessage) []*SensorConfig {
	if sensors, err := NewSensorConfigs(decodeJSONRawSensor(t, testJSON)); err != nil {
		t.Fatalf("Could not parse sensor JSON: %s", err)
	} else {
		if len(sensors) == 0 {
			t.Fatalf("did not get a valid sensor from JSON.")
		}
		return sensors
	}
	return nil
}
