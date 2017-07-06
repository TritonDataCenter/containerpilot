package decoding

import (
	"reflect"
	"testing"
)

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
