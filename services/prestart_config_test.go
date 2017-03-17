package services

import (
	"testing"

	"github.com/joyent/containerpilot/events"
)

func TestPreStartConfigValidate(t *testing.T) {
	preStart, _ := NewPreStartConfig("serviceA", "true 5")
	err := preStart.Validate(nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPreStartConfigBeforeMainService(t *testing.T) {
	preStart, _ := NewPreStartConfig("serviceA", "true 5")
	preStart.Validate(nil)

	mainService := &Config{
		Name:        "main",
		Exec:        []string{"/bin/to/healthcheck/for/service/A.sh", "A1", "A2"},
		ExecTimeout: "100ms",
	}
	mainService.Validate(nil)
	mainService.SetStartup(events.Event{events.ExitSuccess, preStart.Name}, 0)
}
