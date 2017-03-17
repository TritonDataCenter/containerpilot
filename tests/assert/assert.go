package assert

import (
	"reflect"
	"testing"
)

// Equal asserts two interfaces are equal
func Equal(t *testing.T, got, expected interface{}, msg string) {
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(msg, expected, got)
	}
}

// Error asserts we got a particular error
func Error(t *testing.T, err error, expected string) {
	if err == nil {
		t.Fatalf("expected '%s' but got nil error", expected)
	}
	if err.Error() != expected {
		t.Fatalf("expected '%s' but got '%s'", expected, err.Error())
	}
}
