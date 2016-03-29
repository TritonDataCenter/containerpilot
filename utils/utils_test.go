package utils

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"
)

func TestParseCommandArgs(t *testing.T) {
	if cmd, err := ParseCommandArgs(nil); err == nil {
		validateCommandParsed(t, "command", cmd, nil)
	} else {
		t.Errorf("Unexpected parse error: %s", err.Error())
	}

	expected := []string{"/testdata/test.sh", "arg1"}
	json1 := json.RawMessage(`"/testdata/test.sh arg1"`)
	if cmd, err := ParseCommandArgs(json1); err == nil {
		validateCommandParsed(t, "json1", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error json1: %s", err.Error())
	}

	json2 := json.RawMessage(`["/testdata/test.sh","arg1"]`)
	if cmd, err := ParseCommandArgs(json2); err == nil {
		validateCommandParsed(t, "json2", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error json2: %s", err.Error())
	}

	json3 := json.RawMessage(`{ "a": true }`)
	if _, err := ParseCommandArgs(json3); err == nil {
		t.Errorf("Expected parse error for json3")
	}

}

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

func TestParseInterfaces(t *testing.T) {
	if interfaces, err := ParseInterfaces(nil); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if len(interfaces) > 0 {
		t.Errorf("Expected no interfaces, but got %s", interfaces)
	}

	json1 := json.RawMessage(`"eth0"`)
	expected1 := []string{"eth0"}
	if interfaces, err := ParseInterfaces(json1); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if !reflect.DeepEqual(interfaces, expected1) {
		t.Errorf("Expected %s, got: %s", expected1, interfaces)
	}

	json2 := json.RawMessage(`["ethwe","eth0"]`)
	expected2 := []string{"ethwe", "eth0"}
	if interfaces, err := ParseInterfaces(json2); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if !reflect.DeepEqual(interfaces, expected2) {
		t.Errorf("Expected %s, got: %s", expected2, interfaces)
	}

	json3 := json.RawMessage(`{ "a": true }`)
	if _, err := ParseInterfaces(json3); err == nil {
		t.Errorf("Expected parse error for json3")
	}
}
