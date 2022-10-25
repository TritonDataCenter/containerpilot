package telemetry

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/tritondatacenter/containerpilot/tests"
	"github.com/tritondatacenter/containerpilot/tests/mocks"
)

func TestTelemetryConfigParse(t *testing.T) {
	data, _ := os.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRaw(string(data))
	telem, err := NewConfig(testCfg, &mocks.NoopDiscoveryBackend{})
	if err != nil {
		t.Fatalf("could not parse telemetry JSON: %s", err)
	}
	if len(telem.MetricConfigs) != 1 {
		t.Fatalf("expected 1 metric but got %+v", telem.MetricConfigs)
	}
	metric := telem.MetricConfigs[0]
	if _, ok := metric.collector.(prometheus.Counter); !ok {
		t.Fatalf("incorrect collector; expected Counter but got %v", metric.collector)
	}
}

func TestTelemetryConfigBadMetric(t *testing.T) {
	testCfg := tests.DecodeRaw(`{"metrics": [{}], "interfaces": ["inet", "lo0"]}`)
	_, err := NewConfig(testCfg, &mocks.NoopDiscoveryBackend{})
	expected := "invalid metric type"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected '%v' in error from bad metric type but got %v", expected, err)
	}
}

func TestTelemetryConfigBadInterface(t *testing.T) {
	testCfg := tests.DecodeRaw(`{"interfaces": ["xxxx"]}`)
	_, err := NewConfig(testCfg, &mocks.NoopDiscoveryBackend{})
	expected := "none of the interface specifications were able to match"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected '%v' in error from bad metric type but got %v", expected, err)
	}
}
