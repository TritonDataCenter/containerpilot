package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var jsonFragment = []byte(`{
	"port": 8000,
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

func TestTelemetryParse(t *testing.T) {
	if telem, err := NewTelemetry(decodeJSONRawTelemetry(t, jsonFragment)); err != nil {
		t.Fatalf("Could not parse telemetry JSON: %s", err)
	} else {
		if len(telem.Sensors) != 1 {
			t.Fatalf("Expected 1 sensor but got: %v", telem.Sensors)
		}
		sensor := telem.Sensors[0]
		if _, ok := sensor.collector.(prometheus.Counter); !ok {
			t.Fatalf("Incorrect collector; expected Counter but got %v", sensor.collector)
		}
	}
}

func TestTelemetryParseBadSensor(t *testing.T) {
	jsonFragment := []byte(`{"sensors": [{"check": "true"}]}`)
	if _, err := NewTelemetry(decodeJSONRawTelemetry(t, jsonFragment)); err == nil {
		t.Fatalf("Expected error from bad sensor but got nil.")
	} else if ok := strings.HasPrefix(err.Error(), "invalid sensor type"); !ok {
		t.Fatalf("Expected error from bad sensor type but got %v", err)
	}
}

func TestTelemetryParseBadInterface(t *testing.T) {
	jsonFragment := []byte(`{
	"interfaces": ["xxxx"]
 }`)
	if _, err := NewTelemetry(decodeJSONRawTelemetry(t, jsonFragment)); err == nil {
		t.Fatalf("Expected error from bad interface but got nil.")
	} else if ok := strings.HasPrefix(err.Error(), "None of the interface"); !ok {
		t.Fatalf("Expected error from bad interface specification but got %v", err)
	}
}

func decodeJSONRawTelemetry(t *testing.T, testJSON json.RawMessage) interface{} {
	var raw interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("Unexpected error decoding JSON:\n%s\n%v", testJSON, err)
	}
	return raw
}

func TestTelemetryServerRestart(t *testing.T) {
	if telem, err := NewTelemetry(decodeJSONRawTelemetry(t, jsonFragment)); err != nil {
		t.Fatalf("Could not parse telemetry JSON: %s", err)
	} else {
		// initial server
		telem.Serve()
		checkServerIsListening(t, telem)
		telem.Shutdown()

		// reloaded server
		telem, err := NewTelemetry(decodeJSONRawTelemetry(t, jsonFragment))
		if err != nil {
			t.Fatalf("Could not parse telemetry JSON: %s", err)
		}
		telem.Serve()
		checkServerIsListening(t, telem)
	}
}

func checkServerIsListening(t *testing.T, telem *Telemetry) {
	telem.lock.RLock()
	defer telem.lock.RUnlock()
	verifyMetricsEndpointOk(t, telem)
}

func verifyMetricsEndpointOk(t *testing.T, telem *Telemetry) {
	url := fmt.Sprintf("http://%v:%v/metrics", telem.addr.IP, telem.addr.Port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("Got %v status from telemetry server", resp.StatusCode)
	}

}
