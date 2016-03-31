package backends

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"
	"utils"
)

func TestOnChangeCmd(t *testing.T) {
	cmd1 := utils.StrToCmd("./testdata/test.sh doStuff --debug")
	backend := &BackendConfig{
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
	Backends []BackendConfig
}

func TestBackendsParse(t *testing.T) {
	jsonFragment := []byte(`{"backends": [
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
]}`)
	backendFragment := &TestFragmentBackends{}
	if err := json.Unmarshal(jsonFragment, backendFragment); err != nil {
		t.Fatalf("Could not parse backend JSON: %s", err)
	} else {
		backends := backendFragment.Backends
		backend1 := backends[0]
		if err := backend1.Parse(nil); err != nil {
			t.Fatalf("Could not parse backends: %s", err)
		} else {
			validateCommandParsed(t, "onChange", backend1.onChangeCmd,
				[]string{"/bin/to/onChangeEvent/for/upstream/A.sh"})
		}
		backend2 := backends[1]
		if err := backend2.Parse(nil); err != nil {
			t.Fatalf("Could not parse backends: %s", err)
		} else {
			validateCommandParsed(t, "onChange", backend2.onChangeCmd,
				[]string{"/bin/to/onChangeEvent/for/upstream/B.sh"})
		}
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
