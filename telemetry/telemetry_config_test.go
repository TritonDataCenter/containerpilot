package telemetry

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
	"github.com/joyent/containerpilot/tests/mocks"
	"github.com/prometheus/client_golang/prometheus"
)

func TestTelemetryConfigParse(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRaw(string(data))
	telem, err := NewConfig(testCfg, &mocks.NoopDiscoveryBackend{})
	if err != nil {
		t.Fatalf("could not parse telemetry JSON: %s", err)
	}
	assert.Equal(t, len(telem.SensorConfigs), 1, "expected 1 sensor but got: %v")
	sensor := telem.SensorConfigs[0]
	if _, ok := sensor.collector.(prometheus.Counter); !ok {
		t.Fatalf("incorrect collector; expected Counter but got %v", sensor.collector)
	}
}

func TestTelemetryConfigBadSensor(t *testing.T) {
	testCfg := tests.DecodeRaw(`{"sensors": [{"exec": "true", interval: 1}], "interfaces": ["inet"]}`)
	_, err := NewConfig(testCfg, &mocks.NoopDiscoveryBackend{})
	expected := "invalid sensor type"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected '%v' in error from bad sensor type but got %v", expected, err)
	}
}

func TestTelemetryConfigBadInterface(t *testing.T) {
	testCfg := tests.DecodeRaw(`{"interfaces": ["xxxx"]}`)
	_, err := NewConfig(testCfg, &mocks.NoopDiscoveryBackend{})
	expected := "none of the interface specifications were able to match"
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected '%v' in error from bad sensor type but got %v", expected, err)
	}
}
