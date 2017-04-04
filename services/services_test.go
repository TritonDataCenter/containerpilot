package services

import (
	"reflect"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests/mocks"
)

func TestServiceRunSafeClose(t *testing.T) {
	bus := events.NewEventBus()
	ds := mocks.NewDebugSubscriber(bus, 4)
	ds.Run(0)

	cfg := &Config{Name: "myservice", Exec: "true"}
	cfg.Validate(noop)
	svc := NewService(cfg)
	svc.Run(bus)
	svc.Bus.Publish(events.GlobalStartup)
	svc.Close()
	ds.Close()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	svc.Bus.Publish(events.GlobalStartup)

	expected := []events.Event{
		events.GlobalStartup,
		events.Event{events.Stopping, "myservice"},
		events.Event{events.Stopped, "myservice"},
		events.QuitByClose,
	}
	if !reflect.DeepEqual(expected, ds.Results) {
		t.Fatalf("expected: %v\ngot: %v", expected, ds.Results)
	}
}

// A Service should timeout if not started before the startupTimeout
func TestServiceRunStartupTimeout(t *testing.T) {
	bus := events.NewEventBus()
	ds := mocks.NewDebugSubscriber(bus, 5)
	ds.Run(time.Duration(1 * time.Second)) // need to leave room to wait for timeouts

	cfg := &Config{Name: "myservice", Exec: "true"}
	cfg.Validate(noop)
	cfg.setStartup(
		events.Event{events.Startup, "never"},
		time.Duration(100*time.Millisecond),
	)
	svc := NewService(cfg)
	svc.Run(bus)
	svc.Bus.Publish(events.GlobalStartup)

	// note that we can't send a .Close() after this b/c we've timed out
	// and we'll end up blocking forever
	time.Sleep(200 * time.Millisecond)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	ds.Close()

	got := map[events.Event]int{}
	for _, result := range ds.Results {
		got[result]++
	}
	if !reflect.DeepEqual(got, map[events.Event]int{
		events.Event{Code: events.TimerExpired, Source: "myservice"}: 1,
		events.GlobalStartup:                                         1,
		events.QuitByClose:                                           1,
		events.Event{Code: events.Stopping, Source: "myservice"}:     1,
		events.Event{Code: events.Stopped, Source: "myservice"}:      1,
	}) {
		t.Fatalf("expected timeout after startup but got:\n%v", ds.Results)
	}
}

func TestServiceRunRestarts(t *testing.T) {
	log.SetLevel(log.WarnLevel) // test is noisy otherwise

	runRestartsTest := func(restarts interface{}, expected int) {
		bus := events.NewEventBus()
		ds := mocks.NewDebugSubscriber(bus, expected+2) // + start and quit
		ds.Run(time.Duration(100 * time.Millisecond))

		cfg := &Config{
			Name:         "myservice",
			startupEvent: events.GlobalStartup,
			Exec:         []string{"./testdata/test.sh", "doStuff", "runRestartsTest"},
			Restarts:     restarts,
		}
		cfg.Validate(noop)
		svc := NewService(cfg)
		svc.Run(bus)
		svc.Bus.Publish(events.GlobalStartup)
		exitOk := events.Event{Code: events.ExitSuccess, Source: "myservice"}
		var got = 0
		ds.Close()
		for _, result := range ds.Results {
			if result == exitOk {
				got++
			}
		}
		if got != expected {
			t.Fatalf("expected %d restarts but got %d\n%v", expected, got, ds.Results)
		}
	}
	runRestartsTest(3, 4)
	runRestartsTest("1", 2)
	runRestartsTest("never", 1)
	runRestartsTest(0, 1)
	runRestartsTest(nil, 1)
}

func TestServiceRunPeriodic(t *testing.T) {
	//	log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: time.RFC3339Nano})
	log.SetLevel(log.WarnLevel) // test is noisy otherwise
	bus := events.NewEventBus()
	ds := mocks.NewDebugSubscriber(bus, 10)

	cfg := &Config{
		Name:         "myservice",
		startupEvent: events.GlobalStartup,
		Exec:         []string{"./testdata/test.sh", "doStuff", "runPeriodicTest"},
		Frequency:    "10ms",
		Restarts:     "unlimited",
	}
	cfg.Validate(noop)
	svc := NewService(cfg)
	svc.Run(bus)
	ds.Run(time.Duration(100 * time.Millisecond))
	svc.Bus.Publish(events.GlobalStartup)
	exitOk := events.Event{Code: events.ExitSuccess, Source: "myservice"}
	time.Sleep(100 * time.Millisecond)
	svc.Close()
	ds.Close()
	var got = 0
	for _, result := range ds.Results {
		if result == exitOk {
			got++
		}
	}
	if 9 > got || got > 10 {
		t.Fatalf("expected 9 or 10 task fires but got %d\n%v", got, ds.Results)
	}
}

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
