package coprocesses

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCoprocessRestarts(t *testing.T) {

	coprocessRunTest(t, &Coprocess{Restarts: 3}, []byte("...."))
	coprocessRunTest(t, &Coprocess{Restarts: "1"}, []byte(".."))
	coprocessRunTest(t, &Coprocess{Restarts: "never"}, []byte("."))
	coprocessRunTest(t, &Coprocess{Restarts: 0}, []byte("."))
	coprocessRunTest(t, &Coprocess{}, []byte("."))
}

func coprocessRunTest(t *testing.T, coprocess *Coprocess, expected []byte) {
	tmpf, err := ioutil.TempFile("", "gotest")
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	coprocess.Command = []string{"testdata/test.sh", "echoOut", ".", tmpf.Name()}
	err = parseCoprocess(coprocess)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	coprocess.Start()

	// Ensure the task has time to start
	runtime.Gosched()
	ticker := time.NewTicker(1000 * time.Millisecond)
	select {
	case <-ticker.C:
		ticker.Stop()
		coprocess.Stop() // stop if we've taken more than 1 sec
	}
	content, err := ioutil.ReadAll(tmpf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expected, content) {
		t.Errorf("Expected %v but got %v", expected, content)
	}
}

func TestCoprocessParseValidation(t *testing.T) {
	coprocess := &Coprocess{
		Command: []string{"/usr/bin/true"},
		Name:    "",
	}
	expectNoParseError(t, coprocess)
	if coprocess.Name != "/usr/bin/true" {
		t.Errorf("Expected coprocess.Name to be /usr/bin/true but got `%v`",
			coprocess.Name)
	}
	if coprocess.restartLimit != 0 || coprocess.restart != false {
		t.Errorf("Expected coprocess not to have restarts but got: %v",
			coprocess.restartLimit)
	}

	coprocess = &Coprocess{}
	expectParseError(t, coprocess, "Coprocess did not provide a command")
}

func TestCoprocessParseRaw(t *testing.T) {
	expectValues(t, getNew(t, []byte(`[{"command": "true"}]`)), false, 0)
	expectValues(t, getNew(t, []byte(`[{"command": "true", "restarts": 1.1}]`)), true, 1)
	expectValues(t, getNew(t, []byte(`[{"command": "true", "restarts": 1.0}]`)), true, 1)
	expectValues(t, getNew(t, []byte(`[{"command": "true", "restarts": 1}]`)), true, 1)
	expectValues(t, getNew(t, []byte(`[{"command": "true", "restarts": "1"}]`)), true, 1)
}

func TestCoprocessParseRestarts(t *testing.T) {

	const errMsg = `accepts positive integers, "unlimited" or "never"`

	expectParseError(t, &Coprocess{Command: "true", Restarts: "invalid"}, errMsg)
	expectParseError(t, &Coprocess{Command: "true", Restarts: "-1"}, errMsg)
	expectParseError(t, &Coprocess{Command: "true", Restarts: -1}, errMsg)

	expectParsedValues(t, &Coprocess{Command: "true", Restarts: "unlimited"}, true, -2)
	expectParsedValues(t, &Coprocess{Command: "true", Restarts: "never"}, false, 0)
	expectParsedValues(t, &Coprocess{Command: "true", Restarts: 1}, true, 1)
	expectParsedValues(t, &Coprocess{Command: "true", Restarts: "1"}, true, 1)
	expectParsedValues(t, &Coprocess{Command: "true", Restarts: 0}, false, 0)
	expectParsedValues(t, &Coprocess{Command: "true", Restarts: "0"}, false, 0)
	expectParsedValues(t, &Coprocess{Command: "true"}, false, 0)
}

func getNew(t *testing.T, j json.RawMessage) *Coprocess {
	var raw []interface{}
	if err := json.Unmarshal(j, &raw); err != nil {
		t.Fatalf("Unexpected error decoding JSON:\n%s\n%v", j, err)
	} else if coprocesses, err := NewCoprocesses(raw); err != nil {
		t.Fatalf("Expected no errors from %v but got %v", raw, err)
	} else {
		return coprocesses[0]
	}
	return nil
}

func expectParseError(t *testing.T, coprocess *Coprocess, errContains string) {
	if err := parseCoprocess(coprocess); err != nil {
		if !strings.Contains(err.Error(), errContains) {
			t.Errorf("Expected error '%s' but got: %v", errContains, err)
		}
		return
	}
	t.Errorf("Expected error '%s' but didn't get any for %v",
		errContains, coprocess)
}

func expectNoParseError(t *testing.T, coprocess *Coprocess) {
	if err := parseCoprocess(coprocess); err != nil {
		t.Errorf("Unexpected error %v for %v", err, coprocess)
	}
}

func expectParsedValues(t *testing.T, coprocess *Coprocess, doRestart bool, restartLimit int) {
	if err := parseCoprocess(coprocess); err != nil {
		t.Errorf("Unexpected error %v for %v", err, coprocess)
	} else {
		expectValues(t, coprocess, doRestart, restartLimit)
	}
}

func expectValues(t *testing.T, coprocess *Coprocess, doRestart bool, restartLimit int) {
	if coprocess.restart != doRestart {
		t.Errorf("Coprocess.restart was %v but expected %v",
			coprocess.restart, doRestart)
	}
	if coprocess.restartLimit != restartLimit ||
		coprocess.restartsRemain != restartLimit {
		t.Errorf("Coprocess.restartLimit=%v, Coprocess.restartsRemain=%v, but expected %v",
			coprocess.restartLimit, coprocess.restartsRemain, restartLimit)
	}
}
