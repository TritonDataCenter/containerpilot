package services

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

func TestHealthCheck(t *testing.T) {
	cmd1, _ := commands.NewCommand("./testdata/test.sh doStuff --debug", "1s")
	service := &Service{
		healthCheckCmd: cmd1,
	}
	if err := service.CheckHealth(); err != nil {
		t.Errorf("Unexpected error CheckHealth: %s", err)
	}
	// Ensure we can run it more than once
	if err := service.CheckHealth(); err != nil {
		t.Errorf("Unexpected error CheckHealth (x2): %s", err)
	}
}

type TestFragmentServices struct {
	Services []Service
}

func TestServiceParse(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "eth0",
  "health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"]
},
{
  "name": "serviceB",
  "port": 5000,
  "interfaces": ["ethwe","eth0"],
  "health": "/bin/to/healthcheck/for/service/B.sh B1 B2",
  "poll": 30,
  "ttl": 103
}
]`)
	if services, err := NewServices(decodeJSONRawService(t, jsonFragment), nil); err != nil {
		t.Fatalf("Could not parse service JSON: %s", err)
	} else {
		validateCommandParsed(t, "health",
			services[0].healthCheckCmd,
			"/bin/to/healthcheck/for/service/A.sh",
			[]string{"A1", "A2"})
		validateCommandParsed(t, "health",
			services[1].healthCheckCmd,
			"/bin/to/healthcheck/for/service/B.sh",
			[]string{"B1", "B2"})
	}
}

func TestServicesConfigError(t *testing.T) {
	var raw []interface{}
	json.Unmarshal([]byte(`[{"name": ""}]`), &raw)
	_, err := NewServices(raw, nil)
	validateServiceConfigError(t, err, "`name` must not be blank")
	raw = nil

	json.Unmarshal([]byte(`[{"name": "myName"}]`), &raw)
	_, err = NewServices(raw, nil)
	validateServiceConfigError(t, err, "`poll` must be > 0 in service myName")

	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1}]`), &raw)
	_, err = NewServices(raw, nil)
	validateServiceConfigError(t, err, "`ttl` must be > 0 in service myName")

	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1, "ttl": 1}]`), &raw)
	_, err = NewServices(raw, nil)
	validateServiceConfigError(t, err, "`port` must be > 0 in service myName")

	// no health check shouldn't return an error
	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1, "ttl": 1, "port": 80}]`), &raw)
	_, err = NewServices(raw, nil)
	validateServiceConfigError(t, err, "")

	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1, "ttl": 1, "port": 80, "health": "/bin/true", "timeout": "xx"}]`), &raw)
	_, err = NewServices(raw, nil)
	validateServiceConfigError(t, err,
		"Could not parse `health` in service myName: time: invalid duration xx")
}

// ------------------------------------------
// test helpers

func decodeJSONRawService(t *testing.T, testJSON json.RawMessage) []interface{} {
	var raw []interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("Unexpected error decoding JSON:\n%s\n%v", testJSON, err)
	}
	return raw
}

func validateCommandParsed(t *testing.T, name string, parsed *commands.Command,
	expectedExec string, expectedArgs []string) {
	if parsed == nil {
		t.Errorf("%s not configured", name)
	}
	if !reflect.DeepEqual(parsed.Exec, expectedExec) {
		t.Errorf("%s executable not configured: %s != %s", name, parsed.Exec, expectedExec)
	}
	if !reflect.DeepEqual(parsed.Args, expectedArgs) {
		t.Errorf("%s arguments not configured: %s != %s", name, parsed.Args, expectedArgs)
	}
}

func validateServiceConfigError(t *testing.T, err error, expected string) {
	if expected == "" {
		if err != nil {
			t.Fatalf("Expected no error but got %s", err)
		}
	} else {
		if err == nil {
			t.Fatalf("Expected %s but got nil error", expected)
		}
		if err.Error() != expected {
			t.Fatalf("Expected %s but got %s", expected, err.Error())
		}
	}
}
