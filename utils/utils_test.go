package utils

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

func TestToStringArray(t *testing.T) {
	if interfaces, err := ToStringArray(nil); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if len(interfaces) > 0 {
		t.Errorf("Expected no strings, but got %s", interfaces)
	}

	test1 := "eth0"
	expected1 := []string{test1}
	if interfaces, err := ToStringArray(test1); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if !reflect.DeepEqual(interfaces, expected1) {
		t.Errorf("Expected %s, got: %s", expected1, interfaces)
	}

	test2 := []interface{}{"ethwe", "eth0"}
	expected2 := []string{"ethwe", "eth0"}
	if interfaces, err := ToStringArray(test2); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if !reflect.DeepEqual(interfaces, expected2) {
		t.Errorf("Expected %s, got: %s", expected2, interfaces)
	}

	test3 := map[string]interface{}{"a": true}
	if _, err := ToStringArray(test3); err == nil {
		t.Errorf("Expected parse error for json3")
	}
}
