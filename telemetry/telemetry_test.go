package telemetry

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests/mocks"
)

func TestTelemetryServerRestart(t *testing.T) {

	cfg := &Config{Port: 9090, Interfaces: []interface{}{"lo", "lo0", "inet"}}
	cfg.Validate(&mocks.NoopDiscoveryBackend{})

	telem := NewTelemetry(cfg)

	// initial server
	bus := events.NewEventBus()
	telem.Run(bus)
	checkServerIsListening(t, telem)
	telem.Stop()

	// reloaded server
	telem = NewTelemetry(cfg)
	telem.Run(bus)
	checkServerIsListening(t, telem)
}

func checkServerIsListening(t *testing.T, telem *Telemetry) {

	url := fmt.Sprintf("http://%v:%v/metrics", telem.addr.IP, telem.addr.Port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("could not connect to telemetry server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("got %v status from telemetry server", resp.StatusCode)
	}
}
