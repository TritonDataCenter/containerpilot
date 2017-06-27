package jobs

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests/assert"
)

func TestJobRunSafeClose(t *testing.T) {
	bus := events.NewEventBus()
	cfg := &Config{Name: "myjob", Exec: "sleep 10"} // don't want exec to finish
	cfg.Validate(noop)
	job := NewJob(cfg)
	job.Run(bus)
	bus.Publish(events.GlobalStartup)
	job.Quit()
	bus.Wait()
	results := job.Bus.DebugEvents()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	job.Bus.Publish(events.GlobalStartup)

	expected := []events.Event{
		events.GlobalStartup,
		{events.Stopping, "myjob"},
		{events.Stopped, "myjob"},
	}
	if !reflect.DeepEqual(expected, results) {
		t.Fatalf("expected: %v\ngot: %v", expected, results)
	}
}

// A Job should timeout if not started before the startupTimeout
func TestJobRunStartupTimeout(t *testing.T) {
	bus := events.NewEventBus()
	cfg := &Config{Name: "myjob", Exec: "true",
		When: &WhenConfig{Source: "never", Once: "startup", Timeout: "100ms"}}
	cfg.Validate(noop)
	job := NewJob(cfg)
	job.Run(bus)
	job.Bus.Publish(events.GlobalStartup)

	time.Sleep(200 * time.Millisecond)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked but should not: sent to closed Subscriber")
		}
	}()
	bus.Publish(events.QuitByClose)
	job.Quit()
	bus.Wait()
	results := bus.DebugEvents()

	got := map[events.Event]int{}
	for _, result := range results {
		got[result]++
	}
	if !reflect.DeepEqual(got, map[events.Event]int{
		{Code: events.TimerExpired, Source: "myjob"}: 1,
		events.GlobalStartup:                         1,
		{Code: events.Stopping, Source: "myjob"}:     1,
		{Code: events.Stopped, Source: "myjob"}:      1,
		events.QuitByClose:                           1,
	}) {
		t.Fatalf("expected timeout after startup but got:\n%v", results)
	}
}

func TestJobRunRestarts(t *testing.T) {
	runRestartsTest := func(restarts interface{}, expected int) {
		bus := events.NewEventBus()
		cfg := &Config{
			Name:            "myjob",
			whenEvent:       events.GlobalStartup,
			whenStartsLimit: 1,
			Exec:            []string{"./testdata/test.sh", "doStuff", "runRestartsTest"},
			Restarts:        restarts,
		}
		cfg.Validate(noop)
		job := NewJob(cfg)

		job.Run(bus)
		job.Bus.Publish(events.GlobalStartup)
		time.Sleep(100 * time.Millisecond) // TODO: we can't force this, right?
		exitOk := events.Event{Code: events.ExitSuccess, Source: "myjob"}
		var got = 0
		bus.Wait()
		results := bus.DebugEvents()
		for _, result := range results {
			if result == exitOk {
				got++
			}
		}
		if got != expected {
			t.Fatalf("expected %d restarts but got %d\n%v", expected, got, results)
		}
	}
	runRestartsTest(3, 4)
	runRestartsTest("1", 2)
	runRestartsTest("never", 1)
	runRestartsTest(0, 1)
	runRestartsTest(nil, 1)
}

func TestJobRunPeriodic(t *testing.T) {
	bus := events.NewEventBus()

	cfg := &Config{
		Name: "myjob",
		Exec: []string{"./testdata/test.sh", "doStuff", "runPeriodicTest"},
		When: &WhenConfig{Frequency: "250ms"},
		// we need to make sure we don't have any events getting cut off
		// by the test run of 1sec (which would result in flaky tests),
		// so this should ensure we get a predictable number within the window
		Restarts: "2",
	}
	cfg.Validate(noop)
	job := NewJob(cfg)
	job.Run(bus)
	job.Bus.Publish(events.GlobalStartup)
	exitOk := events.Event{Code: events.ExitSuccess, Source: "myjob"}
	exitFail := events.Event{Code: events.ExitFailed, Source: "myjob"}
	time.Sleep(1 * time.Second)
	job.Quit()
	bus.Wait()
	results := bus.DebugEvents()
	var got = 0
	for _, result := range results {
		if result == exitOk {
			got++
		} else {
			if result == exitFail {
				t.Fatalf("no events should have timed-out but got %v", results)
			}
		}
	}
	if got != 3 {
		t.Fatalf("expected exactly 3 task executions but got %d\n%v", got, results)
	}
}

func TestJobMaintenance(t *testing.T) {

	testFunc := func(t *testing.T, startingState JobStatus, event events.Event) JobStatus {
		bus := events.NewEventBus()
		cfg := &Config{Name: "myjob", Exec: "true",
			// need to make sure this can't succeed during test
			Health: &HealthConfig{CheckExec: "false", Heartbeat: 10, TTL: 50},
		}
		cfg.Validate(noop)
		job := NewJob(cfg)
		job.setStatus(startingState)
		job.Run(bus)
		job.Bus.Publish(event)
		job.Quit()
		return job.GetStatus()
	}

	t.Run("enter maintenance", func(t *testing.T) {
		status := testFunc(t, statusUnknown, events.GlobalEnterMaintenance)
		assert.Equal(t, status, statusMaintenance,
			"expected job in '%v' status after entering maintenance but got '%v'")
	})

	// in-flight health checks should not bump the Job out of maintenance
	t.Run("healthy no change", func(t *testing.T) {
		status := testFunc(t, statusMaintenance,
			events.Event{events.ExitSuccess, "check.myjob"})
		assert.Equal(t, status, statusMaintenance,
			"expected job in '%v' status after passing check while in maintenance but got '%v'")
	})

	// in-flight health checks should not bump the Job out of maintenance
	t.Run("unhealthy no change", func(t *testing.T) {
		status := testFunc(t, statusMaintenance,
			events.Event{events.ExitFailed, "check.myjob"})
		assert.Equal(t, status, statusMaintenance,
			"expected job in '%v' status after failed check while in maintenance but got '%v'")
	})

	t.Run("exit maintenance", func(t *testing.T) {
		status := testFunc(t, statusMaintenance, events.GlobalExitMaintenance)
		assert.Equal(t, status, statusUnknown,
			"expected job in '%v' status after exiting maintenance but got '%v'")
	})

	t.Run("now healthy", func(t *testing.T) {
		status := testFunc(t, statusUnknown,
			events.Event{events.ExitSuccess, "check.myjob"})
		assert.Equal(t, status, statusHealthy,
			"expected job in '%v' status after passing check out of maintenance but got '%v'")
	})
}

func TestJobProcessEvent(t *testing.T) {

	t.Run("start each startEvent", func(t *testing.T) {
		// when: {
		//   source: "upstream",
		//   each: "changed"
		// }
		job := &Job{
			Name:         "testJob",
			startEvent:   events.Event{events.StatusChanged, "upstream"},
			startsRemain: unlimited,
			statusLock:   &sync.RWMutex{},
		}
		got := job.processEvent(nil, events.Event{events.StatusChanged, "upstream"})
		assert.False(t, got, "processEvent returned %v after 1st startEvent, expected %v")

		got = job.processEvent(nil, events.Event{events.StatusChanged, "upstream"})
		assert.False(t, got, "processEvent returned %v after 2nd startEvent, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after exit, expected %v")

		got = job.processEvent(nil, events.Event{events.StatusChanged, "upstream"})
		assert.False(t, got, "processEvent returned %v after 3rd startEvent, expected %v")
	})

	t.Run("start one startEvent, with 2 restarts", func(t *testing.T) {
		// when: {
		//   source: "upstream",
		//   once: "changed"
		// },
		// restarts: 2
		job := &Job{
			Name:           "testJob",
			startEvent:     events.Event{events.StatusChanged, "upstream"},
			startsRemain:   1,
			restartLimit:   2,
			restartsRemain: 2,
			statusLock:     &sync.RWMutex{},
		}
		got := job.processEvent(nil, events.Event{events.StatusChanged, "upstream"})
		assert.False(t, got, "processEvent returned %v after 1st startEvent, expected %v")

		got = job.processEvent(nil, events.Event{events.StatusChanged, "upstream"})
		assert.True(t, got, "processEvent returned %v after 2nd startEvent, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after 1st exit, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after 2nd exit, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.True(t, got, "processEvent returned %v after 3rd exit, expected %v")
	})

	t.Run("restart each exit", func(t *testing.T) {
		// restarts: "unlimited"
		job := &Job{
			Name:           "testJob",
			startEvent:     events.GlobalStartup,
			startsRemain:   0,
			restartLimit:   unlimited,
			restartsRemain: unlimited,
			statusLock:     &sync.RWMutex{},
		}
		got := job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after 1st exit, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after 2nd exit, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after 3rd exit, expected %v")
	})

	t.Run("restart once on exit", func(t *testing.T) {
		// restarts: 1
		job := &Job{
			Name:           "testJob",
			startEvent:     events.GlobalStartup,
			startsRemain:   0,
			restartLimit:   1,
			restartsRemain: 1,
			statusLock:     &sync.RWMutex{},
		}
		got := job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.False(t, got, "processEvent returned %v after 1st exit, expected %v")

		got = job.processEvent(nil, events.Event{events.ExitSuccess, "testJob"})
		assert.True(t, got, "processEvent returned %v after 2nd exit, expected %v")
	})

}
