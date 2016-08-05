package backends

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

func TestOnChangeCmd(t *testing.T) {
	cmd1, _ := commands.NewCommand("./testdata/test.sh doStuff --debug", "1s")
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
