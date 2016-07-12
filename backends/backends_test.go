package backends

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

func TestOnChangeCmd(t *testing.T) {
	cmd1 := commands.StrToCmd("./testdata/test.sh doStuff --debug")
	backend := &Backend{
		onChangeCmd: cmd1,
	}
	if _, err := backend.OnChange(); err != nil {
		t.Errorf("Unexpected error OnChange: %s", err)
	}
	// Ensure we can run it more than once
	if _, err := backend.OnChange(); err != nil {
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
  "onChange": "/bin/to/onChangeEvent/for/upstream/A.sh",
  "tag": "dev"
},
{
  "name": "upstreamB",
  "poll": 79,
  "onChange": "/bin/to/onChangeEvent/for/upstream/B.sh"
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
			[]string{"/bin/to/onChangeEvent/for/upstream/A.sh"})
		validateCommandParsed(t, "onChange", backends[1].onChangeCmd,
			[]string{"/bin/to/onChangeEvent/for/upstream/B.sh"})
	}
}

// ------------------------------------------
// test helpers

func validateCommandParsed(t *testing.T, name string, parsed *exec.Cmd, expected []string) {
	if expected == nil {
		if parsed != nil {
			t.Errorf("%s has Cmd, but expected nil", name)
		}
		return
	}
	if parsed == nil {
		t.Errorf("%s not configured", name)
	}
	if parsed.Path != expected[0] {
		t.Errorf("%s path not configured: %s != %s", name, parsed.Path, expected[0])
	}
	if !reflect.DeepEqual(parsed.Args, expected) {
		t.Errorf("%s arguments not configured: %s != %s", name, parsed.Args, expected)
	}
}
