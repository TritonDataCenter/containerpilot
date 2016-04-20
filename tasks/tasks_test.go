package tasks

import (
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
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
