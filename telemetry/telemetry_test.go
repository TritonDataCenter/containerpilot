package telemetry

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"testing"
)

func TestTelemetryParse(t *testing.T) {

	jsonFragment := []byte(`{
	"name": "telemetry",
	"url": "telemetry",
	"port": 8000,
	"ttl": 30,
	"poll": 10,
	"interfaces": ["eth0"],
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

	telemetry := &Telemetry{}
	if err := json.Unmarshal(jsonFragment, &telemetry); err != nil {
		t.Fatalf("Could not parse telemetry JSON: %s", err)
	} else if err := telemetry.Parse(); err != nil {
		t.Fatalf("Could not parse telemetry sensor or interfaces: %s", err)
	}

	if len(telemetry.Sensors) != 1 {
		t.Fatalf("Expected 1 sensor but got: %v", telemetry.Sensors)
	}
	sensor := telemetry.Sensors[0]
	if _, ok := sensor.collector.(prometheus.Counter); !ok {
		t.Fatalf("Incorrect collector; expected Counter but got %v", sensor.collector)
	}
}

func TestTelemetryParseBadSensor(t *testing.T) {
	jsonFragment := []byte(`{"sensors": [{}]}`)
	telemetry := &Telemetry{}
	if err := json.Unmarshal(jsonFragment, &telemetry); err != nil {
		t.Fatalf("Could not parse telemetry JSON: %s", err)
	}
	if err := telemetry.Parse(); err == nil {
		t.Fatalf("Expected error from bad sensor but got nil.")
	} else if ok := strings.HasPrefix(err.Error(), "Invalid sensor type"); !ok {
		t.Fatalf("Expected error from bad sensor type but got %v", err)
	}
}

func TestTelemetryParseBadInterface(t *testing.T) {
	jsonFragment := []byte(`{
	"interfaces": ["xxxx"]
 }`)
	telemetry := &Telemetry{}
	if err := json.Unmarshal(jsonFragment, &telemetry); err != nil {
		t.Fatalf("Could not parse telemetry JSON: %s", err)
	}
	if err := telemetry.Parse(); err == nil {
		t.Fatalf("Expected error from bad interface but got nil.")
	} else if ok := strings.HasPrefix(err.Error(), "None of the interface"); !ok {
		t.Fatalf("Expected error from bad interface specification but got %v", err)
	}
}
