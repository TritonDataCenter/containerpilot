package services

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/discovery"
)

type TestFragmentServices struct {
	Services []Service
}

// Mock Discovery
// TODO this should probably go into the discovery package for use in testing everywhere
type NoopServiceBackend struct{}

func (c *NoopServiceBackend) SendHeartbeat(service *discovery.ServiceDefinition)      { return }
func (c *NoopServiceBackend) CheckForUpstreamChanges(backend, tag string) bool        { return false }
func (c *NoopServiceBackend) MarkForMaintenance(service *discovery.ServiceDefinition) {}
func (c *NoopServiceBackend) Deregister(service *discovery.ServiceDefinition)         {}

func TestServiceConfigValidateExec(t *testing.T) {

	cfg := &ServiceConfig{
		Name:        "serviceA",
		Exec:        []string{"/bin/to/healthcheck/for/service/A.sh", "A1", "A2"},
		ExecTimeout: "1ms",
	}
	assertServiceConfigExecParsed(t, cfg,
		"/bin/to/healthcheck/for/service/A.sh",
		[]string{"A1", "A2"})

	cfg = &ServiceConfig{
		Name:        "serviceB",
		Exec:        "/bin/to/healthcheck/for/service/B.sh B1 B2",
		ExecTimeout: "1ms",
	}
	assertServiceConfigExecParsed(t, cfg,
		"/bin/to/healthcheck/for/service/B.sh",
		[]string{"B1", "B2"})

	cfg = &ServiceConfig{
		Name:        "myName",
		Exec:        "/bin/true",
		ExecTimeout: "xx",
	}
	err := cfg.Validate(&NoopServiceBackend{})
	expected := "could not parse `timeout` for service myName: time: invalid duration xx"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected '%s', got '%v'", expected, err)
	}
}

func TestServicesConfigValidation(t *testing.T) {
	var raw []interface{}
	json.Unmarshal([]byte(`[{"name": ""}]`), &raw)
	_, err := NewServiceConfigs(raw, &NoopServiceBackend{})
	assertServiceConfigError(t, err, "`name` must not be blank")
	raw = nil

	json.Unmarshal([]byte(`[{"name": "myName", "port": 80}]`), &raw)
	_, err = NewServiceConfigs(raw, &NoopServiceBackend{})
	assertServiceConfigError(t, err,
		"`poll` must be > 0 in service `myName` when `port` is set")

	json.Unmarshal([]byte(`[{"name": "myName", "port": 80, "poll": 1}]`), &raw)
	_, err = NewServiceConfigs(raw, &NoopServiceBackend{})
	assertServiceConfigError(t, err,
		"`ttl` must be > 0 in service `myName` when `port` is set")

	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1, "ttl": 1}]`), &raw)
	_, err = NewServiceConfigs(raw, &NoopServiceBackend{})
	assertServiceConfigError(t, err,
		"`heartbeat` and `ttl` may not be set in service `myName` if `port` is not set")

	// no health check shouldn't return an error
	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1, "ttl": 1, "port": 80, "interfaces": "inet"}]`), &raw)
	_, err = NewServiceConfigs(raw, &NoopServiceBackend{})
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
}

func TestServicesConsulExtrasEnableTagOverride(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "inet",
  "health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"],
  "consul": {
	  "enableTagOverride": true
  }
}
]`)

	if services, err := NewServiceConfigs(decodeJSONRawService(t, jsonFragment), nil); err != nil {
		t.Fatalf("could not parse service JSON: %s", err)
	} else {
		if services[0].definition.ConsulExtras.EnableTagOverride != true {
			t.Errorf("ConsulExtras should have had EnableTagOverride set to true.")
		}
	}
}

func TestInvalidServicesConsulExtrasEnableTagOverride(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "inet",
  "health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"],
  "consul": {
	  "enableTagOverride": "nope"
  }
}
]`)

	if _, err := NewServiceConfigs(decodeJSONRawService(t, jsonFragment), nil); err == nil {
		t.Errorf("ConsulExtras should have thrown error about EnableTagOverride being a string.")
	}
}

func TestServicesConsulExtrasDeregisterCriticalServiceAfter(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "inet",
  "health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"],
  "consul": {
	  "deregisterCriticalServiceAfter": "40m"
  }
}
]`)

	if services, err := NewServiceConfigs(decodeJSONRawService(t, jsonFragment), nil); err != nil {
		t.Fatalf("could not parse service JSON: %s", err)
	} else {
		if services[0].definition.ConsulExtras.DeregisterCriticalServiceAfter != "40m" {
			t.Errorf("ConsulExtras should have had DeregisterCriticalServiceAfter set to '40m'.")
		}
	}
}

func TestInvalidServicesConsulExtrasDeregisterCriticalServiceAfter(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "inet",
  "health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"],
  "consul": {
	  "deregisterCriticalServiceAfter": "nope"
  }
}
]`)

	if _, err := NewServiceConfigs(decodeJSONRawService(t, jsonFragment), nil); err == nil {
		t.Errorf("error should have been generated for duration 'nope'.")
	}
}

// ------------------------------------------
// test helpers

func decodeJSONRawService(t *testing.T, testJSON json.RawMessage) []interface{} {
	var raw []interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("unexpected error decoding JSON:\n%s\n%v", testJSON, err)
	}
	return raw
}

func assertServiceConfigExecParsed(t *testing.T, cfg *ServiceConfig,
	expectedExec string, expectedArgs []string) {
	err := cfg.Validate(&NoopServiceBackend{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.exec.Exec, expectedExec) {
		t.Fatalf("executable not configured: %s != %s", cfg.exec.Exec, expectedExec)
	}
	if !reflect.DeepEqual(cfg.exec.Args, expectedArgs) {
		t.Fatalf("arguments not configured: %s != %s", cfg.exec.Args, expectedArgs)
	}
}

func assertServiceConfigError(t *testing.T, err error, expected string) {
	if err == nil {
		t.Fatalf("expected '%s' but got nil error", expected)
	}
	if err.Error() != expected {
		t.Fatalf("expected '%s' but got '%s'", expected, err.Error())
	}
}

func expectRestarts(t *testing.T, cfg *ServiceConfig, doRestart bool, restartLimit int) {
	if cfg.restart != doRestart {
		t.Errorf("service['%v'] restart was %v but expected %v",
			cfg.Name, cfg.restart, doRestart)
	}
	if cfg.restartLimit != restartLimit {
		t.Errorf("service['%v'] restartLimit %v, but expected %v",
			cfg.Name, cfg.restartLimit, restartLimit)
	}
}
