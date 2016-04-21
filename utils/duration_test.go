package utils

import (
	"strings"
	"testing"
	"time"
)

func expectDuration(t *testing.T, in interface{}, expected time.Duration) {
	actual, err := ParseDuration(in)
	if err != nil {
		t.Errorf("Expected %v but got error: %v", expected, err)
	}
	if actual != expected {
		t.Errorf("Expected %v but got %v", expected, actual)
	}
}

func expectError(t *testing.T, in interface{}, errContains string) {
	actual, err := ParseDuration(in)
	if err == nil {
		t.Errorf("Expected error but got: %v", actual)
	}
	if !strings.Contains(err.Error(), errContains) {
		t.Errorf("Expected error '*%s*' but got: %v", errContains, err)
	}
}

func TestParseDuration(t *testing.T) {

	// Bare ints
	expectDuration(t, 1, time.Second)
	expectDuration(t, 10, 10*time.Second)

	// Other ints
	expectDuration(t, int64(10), 10*time.Second)
	expectDuration(t, int32(10), 10*time.Second)
	expectDuration(t, int16(10), 10*time.Second)
	expectDuration(t, int8(10), 10*time.Second)
	expectDuration(t, uint64(10), 10*time.Second)
	expectDuration(t, uint32(10), 10*time.Second)
	expectDuration(t, uint16(10), 10*time.Second)
	expectDuration(t, uint8(10), 10*time.Second)

	// Without Units
	expectDuration(t, "1", time.Second)
	expectDuration(t, "10", 10*time.Second)

	// With Units
	expectDuration(t, "10ns", 10*time.Nanosecond)
	expectDuration(t, "10us", 10*time.Microsecond)
	expectDuration(t, "10ms", 10*time.Millisecond)
	expectDuration(t, "10s", 10*time.Second)
	expectDuration(t, "10m", 10*time.Minute)
	expectDuration(t, "10h", 10*time.Hour)

	// Some parse errors
	expectError(t, "asf", "invalid duration")
	expectError(t, "20yy", "unknown unit yy")

	// Fractional
	expectError(t, 10.10, "unexpected duration of type float")
}
