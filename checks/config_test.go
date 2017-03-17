package checks

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

type TestFragmentChecks struct {
	HealthChecks []HealthCheck
}

func TestCheckParse(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "inet",
  "health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"]
},
{
  "name": "serviceB",
  "port": 5000,
  "interfaces": ["ethwe","eth0", "inet"],
  "health": "/bin/to/healthcheck/for/service/B.sh B1 B2",
  "poll": 30,
  "ttl": 103
}
]`)
	if checks, err := NewConfigs(decodeJSONRawHealthCheck(t, jsonFragment)); err != nil {
		t.Fatalf("could not parse service JSON: %s", err)
	} else {
		validateCommandParsed(t, "health",
			checks[0].exec,
			"/bin/to/healthcheck/for/service/A.sh",
			[]string{"A1", "A2"})
		validateCommandParsed(t, "health",
			checks[1].exec,
			"/bin/to/healthcheck/for/service/B.sh",
			[]string{"B1", "B2"})
	}
}

func TestHealthChecksConfigError(t *testing.T) {
	var raw []interface{}
	json.Unmarshal([]byte(`[{"name": "", "health": "/bin/true"}]`), &raw)
	_, err := NewConfigs(raw)
	validateConfigError(t, err, "`name` must not be blank")
	raw = nil

	json.Unmarshal([]byte(`[{"name": "myName", "health": "/bin/true"}]`), &raw)
	_, err = NewConfigs(raw)
	validateConfigError(t, err, "`poll` must be > 0 in health check myName")

	json.Unmarshal([]byte(`[{"name": "myName", "poll": 1, "ttl": 1, "port": 80, "health": "/bin/true", "timeout": "xx"}]`), &raw)
	_, err = NewConfigs(raw)
	validateConfigError(t, err,
		"could not parse `timeout` in check myName: time: invalid duration xx")
}

// ------------------------------------------
// test helpers

func decodeJSONRawHealthCheck(t *testing.T, testJSON json.RawMessage) []interface{} {
	var raw []interface{}
	if err := json.Unmarshal(testJSON, &raw); err != nil {
		t.Fatalf("unexpected error decoding JSON:\n%s\n%v", testJSON, err)
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

func validateConfigError(t *testing.T, err error, expected string) {
	if expected == "" {
		if err != nil {
			t.Fatalf("expected no error but got '%s'", err)
		}
	} else {
		if err == nil {
			t.Fatalf("expected '%s' but got nil error", expected)
		}
		if err.Error() != expected {
			t.Fatalf("expected '%s' but got '%s'", expected, err.Error())
		}
	}
}
