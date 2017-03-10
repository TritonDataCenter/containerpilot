package telemetry

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/joyent/containerpilot/discovery"
)

func TestTelemetryServerRestart(t *testing.T) {

	cfg := &TelemetryConfig{Port: 9090, Interfaces: []interface{}{"lo", "lo0", "inet"}}
	cfg.Validate(&NoopServiceBackend{})

	telem := NewTelemetry(cfg)
	// initial server
	telem.Serve()
	checkServerIsListening(t, telem)
	telem.Shutdown()

	// reloaded server
	telem = NewTelemetry(cfg)
	telem.Serve()
	checkServerIsListening(t, telem)
}

func checkServerIsListening(t *testing.T, telem *Telemetry) {
	telem.lock.RLock()
	defer telem.lock.RUnlock()
	verifyMetricsEndpointOk(t, telem)
}

func verifyMetricsEndpointOk(t *testing.T, telem *Telemetry) {
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

// Mock Discovery
// TODO this should probably go into the discovery package for use in testing everywhere
type NoopServiceBackend struct{}

func (c *NoopServiceBackend) SendHeartbeat(service *discovery.ServiceDefinition)      { return }
func (c *NoopServiceBackend) CheckForUpstreamChanges(backend, tag string) bool        { return false }
func (c *NoopServiceBackend) MarkForMaintenance(service *discovery.ServiceDefinition) {}
func (c *NoopServiceBackend) Deregister(service *discovery.ServiceDefinition)         {}
