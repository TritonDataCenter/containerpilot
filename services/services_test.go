package services

import (
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
)

func TestServiceRunClose(t *testing.T) {

	bus := events.NewEventBus()
	svc := Service{Name: "myservice"}
	svc.Rx = make(chan events.Event, 1000)
	svc.Flush = make(chan bool)
	svc.startupTimeout = time.Duration(60 * time.Second)
	svc.Run(bus)
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: events.Global})

	svc.Close()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: events.Global}) // should not panic
}

func TestServiceRunTimeout(t *testing.T) {

	bus := events.NewEventBus()
	svc := Service{Name: "myservice"}
	svc.Rx = make(chan events.Event, 1000)
	svc.Flush = make(chan bool)
	svc.startupTimeout = time.Duration(100 * time.Millisecond)
	svc.Run(bus)
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: "serviceA"})

	// note that we can't send a .Close() after this b/c we've timed out
	// and we'll end up blocking forever
	time.Sleep(200 * time.Millisecond)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.Event{Code: events.Startup, Source: "serviceA"}) // should not panic
}

// func TestServiceRunRestarts(t *testing.T) {

// }

// func TestServiceRestarts(t *testing.T) {

// 	coprocessRunTest(t, &CoprocessConfig{Restarts: 3}, []byte("...."))
// 	coprocessRunTest(t, &CoprocessConfig{Restarts: "1"}, []byte(".."))
// 	coprocessRunTest(t, &CoprocessConfig{Restarts: "never"}, []byte("."))
// 	coprocessRunTest(t, &CoprocessConfig{Restarts: 0}, []byte("."))
// 	coprocessRunTest(t, &CoprocessConfig{}, []byte("."))
// }

// func coprocessRunTest(t *testing.T, coprocess *CoprocessConfig, expected []byte) {
// 	tmpf, err := ioutil.TempFile("", "gotest")
// 	defer func() {
// 		tmpf.Close()
// 		os.Remove(tmpf.Name())
// 	}()
// 	if err != nil {
// 		t.Fatalf("Unexpected error: %v", err)
// 	}

// 	coprocess.Exec = []string{"testdata/test.sh", "echoOut", ".", tmpf.Name()}
// 	err = coprocess.Validate()
// 	if err != nil {
// 		t.Fatalf("Unexpected error: %v", err)
// 	}
// 	coprocess.Start()

// 	// Ensure the task has time to start
// 	runtime.Gosched()
// 	ticker := time.NewTicker(1000 * time.Millisecond)
// 	select {
// 	case <-ticker.C:
// 		ticker.Stop()
// 		coprocess.Stop() // stop if we've taken more than 1 sec
// 	}
// 	content, err := ioutil.ReadAll(tmpf)
// 	if err != nil {
// 		t.Fatalf("Unexpected error: %v", err)
// 	}
// 	if !reflect.DeepEqual(expected, content) {
// 		t.Errorf("Expected %v but got %v", expected, content)
// 	}
// }

// func TestTask(t *testing.T) {
// 	tmpf, err := ioutil.TempFile("", "gotest")
// 	defer func() {
// 		tmpf.Close()
// 		os.Remove(tmpf.Name())
// 	}()
// 	if err != nil {
// 		t.Fatalf("Unexpeced error: %v", err)
// 	}
// 	task := &TaskConfig{
// 		Exec:      []string{"testdata/test.sh", "echoOut", ".", tmpf.Name()},
// 		Frequency: "100ms",
// 	}
// 	err = task.Validate()
// 	if err != nil {
// 		t.Fatalf("Unexpeced error: %v", err)
// 	}
// 	// Should print 10 dots (1 per ms)
// 	expected := []byte("..........")
// 	quit := poll(task)
// 	// Ensure the task has time to start
// 	runtime.Gosched()
// 	ticker := time.NewTicker(1050 * time.Millisecond)
// 	select {
// 	case <-ticker.C:
// 		ticker.Stop()
// 		quit <- true
// 	}
// 	content, err := ioutil.ReadAll(tmpf)
// 	if err != nil {
// 		t.Fatalf("Unexpeced error: %v", err)
// 	}
// 	if !reflect.DeepEqual(expected, content) {
// 		t.Errorf("Expected %v but got %v", expected, content)
// 	}
// }

// func TestScheduledTaskTimeoutConfig(t *testing.T) {
// 	tmpf, err := ioutil.TempFile("", "gotest")
// 	defer func() {
// 		tmpf.Close()
// 		os.Remove(tmpf.Name())
// 	}()
// 	if err != nil {
// 		t.Fatalf("Unexpeced error: %v", err)
// 	}
// 	task := &TaskConfig{
// 		Exec:      []string{"testdata/test.sh", "printDots", tmpf.Name()},
// 		Frequency: "400ms",
// 		Timeout:   "200ms",
// 	}
// 	err = task.Validate()
// 	if err != nil {
// 		t.Fatalf("Unexpeced error: %v", err)
// 	}
// 	// Should print 2 dots (timeout 250ms after printing 1 dot every 100ms)
// 	expected := []byte("..")
// 	quit := poll(task)
// 	// Ensure the task has time to start
// 	runtime.Gosched()
// 	// Wait for task to start + 250ms
// 	ticker := time.NewTicker(650 * time.Millisecond)
// 	select {
// 	case <-ticker.C:
// 		ticker.Stop()
// 		quit <- true
// 	}
// 	content, err := ioutil.ReadAll(tmpf)
// 	if err != nil {
// 		t.Fatalf("Unexpected error: %v", err)
// 	}
// 	if !reflect.DeepEqual(expected, content) {
// 		t.Errorf("Expected %s but got %s", expected, content)
// 	}
// }
