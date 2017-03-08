package checks

import (
	"context"
	"testing"
	"time"

	"github.com/joyent/containerpilot/commands"
)

func TestHealthCheck(t *testing.T) {
	cmd1, _ := commands.NewCommand("./testdata/test.sh doStuff --debug",
		time.Duration(1*time.Second))
	check := &HealthCheck{exec: cmd1}
	if err := check.CheckHealth(context.Background()); err != nil {
		t.Errorf("Unexpected error CheckHealth: %s", err)
	}
	// Ensure we can run it more than once
	if err := check.CheckHealth(context.Background()); err != nil {
		t.Errorf("Unexpected error CheckHealth (x2): %s", err)
	}
}

func TestHealthCheckBad(t *testing.T) {
	cmd1, _ := commands.NewCommand("./testdata/test.sh failStuff",
		time.Duration(1*time.Second))
	check := &HealthCheck{exec: cmd1}
	if err := check.CheckHealth(context.Background()); err == nil {
		t.Errorf("Expected error from CheckHealth but got nil")
	}
}
