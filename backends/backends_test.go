package backends

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

func TestOnChangeCmd(t *testing.T) {
	cmd1, _ := commands.NewCommand("./testdata/test.sh doStuff --debug", "1s", nil)
	backend := &Backend{
		onChangeCmd: cmd1,
	}
	if err := backend.OnChange(); err != nil {
		t.Errorf("Unexpected error OnChange: %s", err)
	}
	// Ensure we can run it more than once
	if err := backend.OnChange(); err != nil {
		t.Errorf("Unexpected error OnChange (x2): %s", err)
	}
}

type TestFragmentBackends struct {
	Backends []Backend
}

func TestBackendsParse(t *testing.T) {
	jsonFragment := []byte(`[
{
  "name": "upstreamA",
  "poll": 11,
  "onChange": ["/bin/to/onChangeEvent/for/upstream/A.sh", "A1", "A2"],
  "tag": "dev"
},
{
  "name": "upstreamB",
  "poll": 79,
  "onChange": "/bin/to/onChangeEvent/for/upstream/B.sh B1 B2"
}
]`)

	var raw []interface{}
	if err := json.Unmarshal(jsonFragment, &raw); err != nil {
		t.Fatalf("Unexpected error decoding JSON: %v", err)
	}

	if backends, err := NewBackends(raw, nil); err != nil {
		t.Fatalf("Could not parse backends JSON: %s", err)
	} else {
		validateCommandParsed(t, "onChange", backends[0].onChangeCmd,
			"/bin/to/onChangeEvent/for/upstream/A.sh",
			[]string{"A1", "A2"})
		validateCommandParsed(t, "onChange", backends[1].onChangeCmd,
			"/bin/to/onChangeEvent/for/upstream/B.sh",
			[]string{"B1", "B2"})
	}
}

func TestBackendsConfigError(t *testing.T) {
	var raw []interface{}
	json.Unmarshal([]byte(`[{"name": ""}]`), &raw)
	_, err := NewBackends(raw, nil)
	validateBackendConfigError(t, err, "`name` must not be blank")
	raw = nil

	json.Unmarshal([]byte(`[{"name": "myName"}]`), &raw)
	_, err = NewBackends(raw, nil)
	validateBackendConfigError(t, err, "`onChange` is required in backend myName")

	json.Unmarshal([]byte(`[{"name": "myName", "onChange": "/bin/true", "timeout": "xx"}]`), &raw)
	_, err = NewBackends(raw, nil)
	validateBackendConfigError(t, err,
		"Could not parse `onChange` in backend myName: time: invalid duration xx")

	json.Unmarshal([]byte(`[{"name": "myName", "onChange": "/bin/true", "timeout": ""}]`), &raw)
	_, err = NewBackends(raw, nil)
	validateBackendConfigError(t, err, "`poll` must be > 0 in backend myName")
}

// ------------------------------------------
// test helpers

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

func validateBackendConfigError(t *testing.T, err error, expected string) {
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
