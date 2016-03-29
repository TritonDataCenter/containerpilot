package metrics

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"strings"
	"testing"
)

func TestMetricsParse(t *testing.T) {

	jsonFragment := []byte(`{
	"name": "namespace_text",
	"url": "subsystem_text",
	"port": 8000,
	"ttl": 30,
	"poll": 10,
	"interfaces": ["eth0"],
	"sensors": [
       {
		"namespace": "namespace_text",
		"subsystem": "subsystem_text",
		"name": "sensor_name_metrics_test",
		"help": "help text",
		"type": "counter",
		"poll": 5,
		"check": ["/bin/sensor.sh"]
	  }
	]
 }`)

	metrics := &Metrics{}
	if err := json.Unmarshal(jsonFragment, &metrics); err != nil {
		t.Fatalf("Could not parse metrics JSON: %s", err)
	} else if err := metrics.Parse(); err != nil {
		t.Fatalf("Could not parse metrics sensor or interfaces: %s", err)
	}

	if len(metrics.Sensors) != 1 {
		t.Fatalf("Expected 1 sensor but got: %v", metrics.Sensors)
	}
	sensor := metrics.Sensors[0]
	if _, ok := sensor.collector.(prometheus.Counter); !ok {
		t.Fatalf("Incorrect collector; expected Counter but got %v", sensor.collector)
	}
}

func TestMetricsParseBadSensor(t *testing.T) {
	jsonFragment := []byte(`{"sensors": [{}]}`)
	metrics := &Metrics{}
	if err := json.Unmarshal(jsonFragment, &metrics); err != nil {
		t.Fatalf("Could not parse metrics JSON: %s", err)
	}
	if err := metrics.Parse(); err == nil {
		t.Fatalf("Expected error from bad sensor but got nil.")
	} else if ok := strings.HasPrefix(err.Error(), "Invalid sensor type"); !ok {
		t.Fatalf("Expected error from bad sensor type but got %v", err)
	}
}

func TestMetricsParseBadInterface(t *testing.T) {
	jsonFragment := []byte(`{
	"interfaces": ["xxxx"]
 }`)
	metrics := &Metrics{}
	if err := json.Unmarshal(jsonFragment, &metrics); err != nil {
		t.Fatalf("Could not parse metrics JSON: %s", err)
	}
	if err := metrics.Parse(); err == nil {
		t.Fatalf("Expected error from bad interface but got nil.")
	} else if ok := strings.HasPrefix(err.Error(), "None of the interface"); !ok {
		t.Fatalf("Expected error from bad interface specification but got %v", err)
	}
}
