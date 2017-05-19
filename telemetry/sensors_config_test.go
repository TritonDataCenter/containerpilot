package telemetry

import (
	"fmt"
	"testing"

	"github.com/joyent/containerpilot/tests"
	"github.com/prometheus/client_golang/prometheus"
)

func TestSensorConfigParse(t *testing.T) {

	errMsg := "incorrect collector; expected %v but got %v"
	fragment := `[{
	namespace: "telemetry",
	subsystem: "sensors",
	name: "TestSensorConfigParse",
	help: "help",
	type: "%s"
}]`

	testCfg := tests.DecodeRawToSlice(fmt.Sprintf(fragment, "counter"))
	sensors, _ := NewSensorConfigs(testCfg)
	collector := sensors[0].collector
	if _, ok := collector.(prometheus.Counter); !ok {
		t.Fatalf(errMsg, collector, "Counter")
	}

	testCfg = tests.DecodeRawToSlice(fmt.Sprintf(fragment, "gauge"))
	sensors, _ = NewSensorConfigs(testCfg)
	collector = sensors[0].collector
	if _, ok := collector.(prometheus.Gauge); !ok {
		t.Fatalf(errMsg, collector, "Gauge")
	}

	testCfg = tests.DecodeRawToSlice(fmt.Sprintf(fragment, "histogram"))
	sensors, _ = NewSensorConfigs(testCfg)
	collector = sensors[0].collector
	if _, ok := collector.(prometheus.Histogram); !ok {
		t.Fatalf(errMsg, collector, "Histogram")
	}

	testCfg = tests.DecodeRawToSlice(fmt.Sprintf(fragment, "summary"))
	sensors, _ = NewSensorConfigs(testCfg)
	collector = sensors[0].collector
	if _, ok := collector.(prometheus.Summary); !ok {
		t.Fatalf(errMsg, collector, "Summary")
	}
}

// invalid collector type
func TestSensorConfigBadType(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[{
	namespace: "telemetry",
	subsystem: "sensors",
	name: "TestSensorBadType",
	type: "nonsense"}]`)

	if sensors, err := NewSensorConfigs(testCfg); err == nil {
		t.Fatalf("did not get expected error from parsing sensors: %v", sensors)
	}
}

// invalid metric name
func TestSensorConfigBadName(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[{
	"namespace": "telemetry",
	"subsystem": "sensors",
	"name": "Test.Sensor.Bad.Name",
	"type": "counter"}]`)

	if sensors, err := NewSensorConfigs(testCfg); err == nil {
		t.Fatalf("did not get expected error from parsing sensors: %v", sensors)
	}
}

// partial metric name parses ok and write out as expected
func TestSensorConfigPartialName(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[{
	"name": "telemetry_sensors_partial_name",
	"help": "help text",
	"type": "counter"}]`)

	sensors, _ := NewSensorConfigs(testCfg)
	if _, ok := sensors[0].collector.(prometheus.Counter); !ok {
		t.Fatalf("incorrect collector; expected Counter but got %v", sensors[0].collector)
	}
}
