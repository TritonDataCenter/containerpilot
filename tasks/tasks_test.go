package tasks

import "testing"
import "time"
import "io/ioutil"
import "os"
import "reflect"

func TestTaskConfig(t *testing.T) {
	tmpf, err := ioutil.TempFile("", "gotest")
	defer func() {
		tmpf.Close()
		os.Remove(tmpf.Name())
	}()
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	task := &TaskConfig{
		Args:      []string{"testdata/test.sh", "echoOut", ".", tmpf.Name()},
		Frequency: "100ms",
	}
	err = parseTask(task)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	// Should print 10 dots (1 per ms)
	expected := []byte("..........")
	task.Start()
	ticker := time.NewTicker(1050 * time.Millisecond)
	select {
	case <-ticker.C:
		task.Stop()
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
	task := &TaskConfig{
		Args:      []string{"testdata/test.sh", "printDots", tmpf.Name()},
		Frequency: "1000ms",
		Timeout:   "200ms",
	}
	err = parseTask(task)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	// Should print 2 dots (timeout 250ms after printing 1 dot every 100ms)
	expected := []byte("..")
	task.Start()
	ticker := time.NewTicker(1050 * time.Millisecond)
	select {
	case <-ticker.C:
		task.Stop()
	}
	content, err := ioutil.ReadAll(tmpf)
	if err != nil {
		t.Fatalf("Unexpeced error: %v", err)
	}
	if !reflect.DeepEqual(expected, content) {
		t.Errorf("Expected %v but got %v", expected, content)
	}
}
