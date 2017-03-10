package telemetry

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var testConfig = []byte(`{
	"port": 8000,
	"interfaces": ["inet"],
	"sensors": [
       {
		"namespace": "telemetry",
		"subsystem": "telemetry",
		"name": "TestTelemetryParse",
		"help": "help",
		"type": "counter",
		"poll": 5,
		"check": ["/bin/sensor.sh"]
	  }
	]
 }`)

func TestTelemetryConfigParse(t *testing.T) {
	jsonFragment := decodeRaw(t, testConfig)
	if telem, err := NewTelemetryConfig(jsonFragment, &NoopServiceBackend{}); err != nil {
		t.Fatalf("could not parse telemetry JSON: %s", err)
	} else {
		if len(telem.SensorConfigs) != 1 {
			t.Fatalf("expected 1 sensor but got: %v", telem.SensorConfigs)
		}
		sensor := telem.SensorConfigs[0]
		if _, ok := sensor.collector.(prometheus.Counter); !ok {
			t.Fatalf("incorrect collector; expected Counter but got %v", sensor.collector)
		}
	}
}

func TestTelemetryConfigBadSensor(t *testing.T) {
	raw := []byte(`{"sensors": [{"check": "true"}], "interfaces": ["inet"]}`)
	jsonFragment := decodeRaw(t, raw)
	_, err := NewTelemetryConfig(jsonFragment, &NoopServiceBackend{})
	expected := "invalid sensor type"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected '%v' in error from bad sensor type but got %v", err)
	}
}

func TestTelemetryConfigBadInterface(t *testing.T) {
	jsonFragment := decodeRaw(t, []byte(`{"interfaces": ["xxxx"]}`))
	_, err := NewTelemetryConfig(jsonFragment, &NoopServiceBackend{})
	expected := "none of the interface specifications were able to match"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected '%v' in error from bad interface spec but got '%v'", expected, err)
	}
}

// test helper for making sure our test JSON is actually parsable
func decodeRaw(t *testing.T, testJSON json.RawMessage) interface{} {
	var raw interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("unexpected error decoding JSON:\n%s\n%v", testJSON, err)
	}
	return raw
}
