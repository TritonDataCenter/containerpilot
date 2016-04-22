package tasks

import (
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// We can't use the poll function defined in core/poll.go
// because it results in an import cycle
func poll(task *Task) chan bool {
	ticker := time.NewTicker(task.PollTime())
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				task.PollAction()
			case <-quit:
				task.PollStop()
				return
			}
		}
	}()
	return quit
}

func expectParseError(t *testing.T, task *Task, errContains string) {
	if err := parseTask(task); err != nil {
		if !strings.Contains(err.Error(), errContains) {
			t.Errorf("Expected error '%s' but got: %v", errContains, err)
		}
		return
	}
	t.Errorf("Expected error '%s' but didn't get any", errContains)
}

func expectNoParseError(t *testing.T, task *Task) {
	if err := parseTask(task); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func expectDuration(t *testing.T, actual time.Duration, expectedString string) {
	expected, err := time.ParseDuration(expectedString)
	if err != nil {
		t.Fatalf("expectedString '%s' failed to parse: %v", expectedString, err)
	}
	if expected != actual {
		t.Errorf("expected duration %v but got %v", expected, actual)
	}
}

func TestTaskParseDuration(t *testing.T) {
	task := &Task{
		Args:      []string{"/usr/bin/true"},
		Frequency: "1ms",
	}
	expectNoParseError(t, task)
	expectDuration(t, task.freqDuration, "1ms")
	expectDuration(t, task.timeoutDuration, "1ms")

	task = &Task{
		Args:      []string{"/usr/bin/true"},
		Frequency: "10s",
		Timeout:   "10",
	}
	expectNoParseError(t, task)
	expectDuration(t, task.freqDuration, "10s")
	expectDuration(t, task.timeoutDuration, "10s")

	task = &Task{
		Args:      []string{"/usr/bin/true"},
		Frequency: "10",
		Timeout:   "1s",
	}
	expectNoParseError(t, task)
	expectDuration(t, task.freqDuration, "10s")
	expectDuration(t, task.timeoutDuration, "1s")

	task.Frequency = "-1ms"
	expectParseError(t, task, "Frequency -1ms cannot be less that 1ms")

	task.Frequency = "1ns"
	expectParseError(t, task, "Frequency 1ns cannot be less that 1ms")

	task.Frequency = "1ms"
	task.Timeout = "-1ms"
	expectParseError(t, task, "Timeout -1ms cannot be less that 1ms")

	task.Timeout = "1ns"
	expectParseError(t, task, "Timeout 1ns cannot be less that 1ms")
}

func TestTask(t *testing.T) {
	tmpf, err := ioutil.TempFile("", "gotest")
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	task := &Task{
		Args:      []string{"testdata/test.sh", "echoOut", ".", tmpf.Name()},
		Frequency: "100ms",
	}
	err = parseTask(task)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	// Should print 10 dots (1 per ms)
	expected := []byte("..........")
	quit := poll(task)
	// Ensure the task has time to start
	runtime.Gosched()
	ticker := time.NewTicker(1050 * time.Millisecond)
	select {
	case <-ticker.C:
		ticker.Stop()
		quit <- true
	}
	content, err := ioutil.ReadAll(tmpf)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	if !reflect.DeepEqual(expected, content) {
		t.Errorf("Expected %v but got %v", expected, content)
	}
}

func TestScheduledTaskTimeoutConfig(t *testing.T) {
	tmpf, err := ioutil.TempFile("", "gotest")
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	task := &Task{
		Args:      []string{"testdata/test.sh", "printDots", tmpf.Name()},
		Frequency: "400ms",
		Timeout:   "200ms",
	}
	err = parseTask(task)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	// Should print 2 dots (timeout 250ms after printing 1 dot every 100ms)
	expected := []byte("..")
	quit := poll(task)
	// Ensure the task has time to start
	runtime.Gosched()
	// Wait for task to start + 250ms
	ticker := time.NewTicker(650 * time.Millisecond)
	select {
	case <-ticker.C:
		ticker.Stop()
		quit <- true
	}
	content, err := ioutil.ReadAll(tmpf)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	if !reflect.DeepEqual(expected, content) {
		t.Errorf("Expected %s but got %s", expected, content)
	}
}
