package services

import (
	"testing"
	"time"
)

func TestTaskValidateDuration(t *testing.T) {
	task := &TaskConfig{
		Exec:      []string{"/usr/bin/true"},
		Frequency: "1ms",
	}
	expectTaskValidationNoError(t, task)
	expectTaskValidationDuration(t, task.freqDuration, "1ms")
}

func TestTaskValidateTimeout(t *testing.T) {
	task := &TaskConfig{
		Exec:      []string{"/usr/bin/true"},
		Frequency: "10s",
		Timeout:   "10",
	}
	expectTaskValidationNoError(t, task)
	expectTaskValidationDuration(t, task.freqDuration, "10s")
}

func TestTaskValidateFrequency(t *testing.T) {
	task := &TaskConfig{
		Exec:      []string{"/usr/bin/true"},
		Frequency: "10",
		Timeout:   "1s",
	}
	expectTaskValidationNoError(t, task)
	expectTaskValidationDuration(t, task.freqDuration, "10s")

	task.Frequency = "-1ms"
	expectTaskValidationError(t, task, "frequency -1ms cannot be less that 1ms")

	task.Frequency = "1ns"
	expectTaskValidationError(t, task, "frequency 1ns cannot be less that 1ms")

	task.Frequency = "1ms"
	task.Timeout = "-1ms"
	expectTaskValidationError(t, task, "timeout -1ms cannot be less that 1ms")

	task.Timeout = "1ns"
	expectTaskValidationError(t, task, "timeout 1ns cannot be less that 1ms")
}

// test helper functions

func expectTaskValidationNoError(t *testing.T, task *TaskConfig) {
	if err := task.Validate(); err != nil {
		t.Fatalf("unexpected error in task.Validate(): %v", err)
	}
}

func expectTaskValidationError(t *testing.T, task *TaskConfig, expectedErr string) {
	err := task.Validate()
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("expected error '%s' but got: %v", expectedErr, err)
	}
}

func expectTaskValidationDuration(t *testing.T, actual time.Duration, expectedString string) {
	expected, err := time.ParseDuration(expectedString)
	if err != nil {
		t.Fatalf("expected duration '%s' failed to parse: '%v'", expectedString, err)
	}
	if expected != actual {
		t.Fatalf("expected duration '%v' but got '%v'", expected, actual)
	}
}
