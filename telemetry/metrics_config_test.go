package telemetry

import (
	"fmt"
	"testing"

	"github.com/tritondatacenter/containerpilot/tests"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricConfigParse(t *testing.T) {

	errMsg := "incorrect collector; expected %v but got %v"
	fragment := `[{
	namespace: "telemetry",
	subsystem: "metrics",
	name: "TestMetricConfigParse",
	help: "help",
	type: "%s"
}]`

	testCfg := tests.DecodeRawToSlice(fmt.Sprintf(fragment, "counter"))
	metrics, _ := NewMetricConfigs(testCfg)
	collector := metrics[0].collector
	if _, ok := collector.(prometheus.Counter); !ok {
		t.Fatalf(errMsg, collector, "Counter")
	}

	testCfg = tests.DecodeRawToSlice(fmt.Sprintf(fragment, "gauge"))
	metrics, _ = NewMetricConfigs(testCfg)
	collector = metrics[0].collector
	if _, ok := collector.(prometheus.Gauge); !ok {
		t.Fatalf(errMsg, collector, "Gauge")
	}

	testCfg = tests.DecodeRawToSlice(fmt.Sprintf(fragment, "histogram"))
	metrics, _ = NewMetricConfigs(testCfg)
	collector = metrics[0].collector
	if _, ok := collector.(prometheus.Histogram); !ok {
		t.Fatalf(errMsg, collector, "Histogram")
	}

	testCfg = tests.DecodeRawToSlice(fmt.Sprintf(fragment, "summary"))
	metrics, _ = NewMetricConfigs(testCfg)
	collector = metrics[0].collector
	if _, ok := collector.(prometheus.Summary); !ok {
		t.Fatalf(errMsg, collector, "Summary")
	}
}

// invalid collector type
func TestMetricConfigBadType(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[{
	namespace: "telemetry",
	subsystem: "metrics",
	name: "TestMetricBadType",
	type: "nonsense"}]`)

	if metrics, err := NewMetricConfigs(testCfg); err == nil {
		t.Fatalf("did not get expected error from parsing metrics: %v", metrics)
	}
}

// invalid metric name
func TestMetricConfigBadName(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[{
	"namespace": "telemetry",
	"subsystem": "metrics",
	"name": "Test.Metric.Bad.Name",
	"type": "counter"}]`)

	if metrics, err := NewMetricConfigs(testCfg); err == nil {
		t.Fatalf("did not get expected error from parsing metrics: %v", metrics)
	}
}

// partial metric name parses ok and write out as expected
func TestMetricConfigPartialName(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[{
	"name": "telemetry_metrics_partial_name",
	"help": "help text",
	"type": "counter"}]`)

	metrics, _ := NewMetricConfigs(testCfg)
	if _, ok := metrics[0].collector.(prometheus.Counter); !ok {
		t.Fatalf("incorrect collector; expected Counter but got %v", metrics[0].collector)
	}
}
