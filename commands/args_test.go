package commands

import (
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
	if cmd, err := ParseCommandArgs("/testdata/test.sh arg1"); err == nil {
		validateCommandParsed(t, "string", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error string: %s", err.Error())
	}

	if cmd, err := ParseCommandArgs([]string{"/testdata/test.sh", "arg1"}); err == nil {
		validateCommandParsed(t, "[]string", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error []string: %s", err.Error())
	}

	if cmd, err := ParseCommandArgs([]interface{}{"/testdata/test.sh", "arg1"}); err == nil {
		validateCommandParsed(t, "[]interface{}", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error []interface{}: %s", err.Error())
	}

	if _, err := ParseCommandArgs(map[string]bool{"a": true}); err == nil {
		t.Errorf("Expected parse error for invalid")
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
