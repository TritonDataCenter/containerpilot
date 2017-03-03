package watches

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joyent/containerpilot/commands"
)

type TestFragmentWatchs struct {
	Watchs []Watch
}

func TestWatchsParse(t *testing.T) {
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
		t.Fatalf("unexpected error decoding JSON: %v", err)
	}

	if watchs, err := NewWatches(raw, nil); err != nil {
		t.Fatalf("could not parse watchs JSON: %s", err)
	} else {
		validateCommandParsed(t, "onChange", watchs[0].exec,
			"/bin/to/onChangeEvent/for/upstream/A.sh",
			[]string{"A1", "A2"})
		validateCommandParsed(t, "onChange", watchs[1].exec,
			"/bin/to/onChangeEvent/for/upstream/B.sh",
			[]string{"B1", "B2"})
	}
}

func TestWatchsConfigError(t *testing.T) {
	var raw []interface{}
	json.Unmarshal([]byte(`[{"name": ""}]`), &raw)
	_, err := NewWatches(raw, nil)
	validateWatchConfigError(t, err, "`name` must not be blank")
	raw = nil

	json.Unmarshal([]byte(`[{"name": "myName"}]`), &raw)
	_, err = NewWatches(raw, nil)
	validateWatchConfigError(t, err, "`onChange` is required in watch myName")

	json.Unmarshal([]byte(`[{"name": "myName", "onChange": "/bin/true", "poll": 1, "timeout": "xx"}]`), &raw)
	_, err = NewWatches(raw, nil)
	validateWatchConfigError(t, err,
		"could not parse `onChange` in watch myName: time: invalid duration xx")

	json.Unmarshal([]byte(`[{"name": "myName", "onChange": "/bin/true", "timeout": ""}]`), &raw)
	_, err = NewWatches(raw, nil)
	validateWatchConfigError(t, err, "`poll` must be > 0 in watch myName")
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

func validateWatchConfigError(t *testing.T, err error, expected string) {
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
