package services

import (
	"testing"
	"time"
)

func TestTaskValidateDuration(t *testing.T) {
	task := &Config{
		Exec:      []string{"/usr/bin/true"},
		Frequency: "1ms",
	}
	expectTaskValidationNoError(t, task)
	expectTaskValidationDuration(t, task.freqInterval, time.Millisecond)
}

func TestTaskValidateTimeout(t *testing.T) {
	task := &Config{
		Exec:        []string{"/usr/bin/true"},
		Frequency:   "10s",
		ExecTimeout: "10",
	}
	expectTaskValidationNoError(t, task)
	expectTaskValidationDuration(t, task.freqInterval, time.Second*10)
}

func TestTaskValidateFrequency(t *testing.T) {
	task := &Config{
		Exec:        []string{"/usr/bin/true"},
		Frequency:   "10",
		ExecTimeout: "1s",
	}
	expectTaskValidationNoError(t, task)
	expectTaskValidationDuration(t, task.freqInterval, time.Second*10)

	task.Frequency = "-1ms"
	expectTaskValidationError(t, task, "frequency '-1ms' cannot be less than 1ms")

	task.Frequency = "1ns"
	expectTaskValidationError(t, task, "frequency '1ns' cannot be less than 1ms")

	task.Frequency = "1ms"
	task.ExecTimeout = "-1ms"
	expectTaskValidationError(t, task, "timeout '-1ms' cannot be less than 1ms")

	task.ExecTimeout = "1ns"
	expectTaskValidationError(t, task, "timeout '1ns' cannot be less than 1ms")
}

// test helper functions

func expectTaskValidationNoError(t *testing.T, task *Config) {
	if err := task.Validate(nil); err != nil {
		t.Fatalf("unexpected error in task.Validate(): %v", err)
	}
}

func expectTaskValidationError(t *testing.T, task *Config, expectedErr string) {
	err := task.Validate(nil)
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("expected error '%s' but got: %v", expectedErr, err)
	}
}

func expectTaskValidationDuration(t *testing.T, actual time.Duration, expected time.Duration) {
	if expected != actual {
		t.Fatalf("expected duration '%v' but got '%v'", expected, actual)
	}
}
