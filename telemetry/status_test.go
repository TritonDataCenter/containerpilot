package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tritondatacenter/containerpilot/jobs"
	"github.com/tritondatacenter/containerpilot/tests"
	"github.com/tritondatacenter/containerpilot/tests/mocks"
	"github.com/tritondatacenter/containerpilot/watches"
)

func TestStatusServerPostInvalid(t *testing.T) {
	cfg := &Config{Port: 9090, Interfaces: []interface{}{"lo", "lo0", "inet"}}
	cfg.Validate(&mocks.NoopDiscoveryBackend{})
	telem := NewTelemetry(cfg)
	ctx := context.Background()
	defer telem.Stop(ctx)
	telem.Run(ctx)
	url := fmt.Sprintf("http://%v:%v/status", telem.addr.IP, telem.addr.Port)
	resp, err := http.Post(url, "", nil)
	if err != nil {
		t.Fatalf("could not connect to status endpoint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 405 {
		t.Fatalf("expected Method Not Allowed from /status endpoint but got %v",
			resp.StatusCode)
	}
}

func TestStatusServerGet(t *testing.T) {
	noop := &mocks.NoopDiscoveryBackend{}
	var err error

	jobCfgs, err := jobs.NewConfigs(
		tests.DecodeRawToSlice(
			`[
				{
					name: "myjob1",
					exec: "sleep 10"
				},
				{
					name: "myjob2",
					exec: "sleep 10",
					port: 80,
					interfaces: ["inet", "lo0"],
					health: {
					  exec: "true",
					  interval: 1,
					  ttl: 10
					}
				},
				{
					name: "myjob3",
					exec: "sleep 10",
					health: {
					  exec: "true",
					  interval: 1,
					  ttl: 10
					}
				}
			]`),
		noop)
	if err != nil {
		t.Fatal(err)
	}
	jobs := jobs.FromConfigs(jobCfgs)

	watchCfgs, err := watches.NewConfigs(
		tests.DecodeRawToSlice(
			`[{name: "watch1", interval: 1},
			 {name: "watch2", interval: 2}]`),
		noop)
	if err != nil {
		t.Fatal(err)
	}
	watches := watches.FromConfigs(watchCfgs)
	cfg := &Config{Port: 9090, Interfaces: []interface{}{"lo", "lo0", "inet"}}
	cfg.Validate(&mocks.NoopDiscoveryBackend{})

	telem := NewTelemetry(cfg)
	telem.MonitorJobs(jobs)
	telem.MonitorWatches(watches)

	ctx := context.Background()
	defer telem.Stop(ctx)
	telem.Run(ctx)

	url := fmt.Sprintf("http://%v:%v/status", telem.addr.IP, telem.addr.Port)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("could not connect to status endpoint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("got %v status from status endpoint", resp.StatusCode)
	}

	// parse and check the response body
	var out Status
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		t.Fatal(err)
	}

	// expected, actual
	assert.Equal(t, []string{"watch1", "watch2"}, out.Watches,
		"unexpected value for 'watches'")
	assert.Equal(t, 1, len(out.Services), "unexpected count of services")
	assert.Equal(t, 80, out.Services[0].Port, "unexpected job port")
	assert.Equal(t, "unknown", out.Services[0].Status, "unexpected job status")
	assert.Equal(t, 2, len(out.Jobs), "unexpected count of services")
	assert.Equal(t, "myjob1", out.Jobs[0].Name)
	assert.Equal(t, "unknown", out.Jobs[0].Status, "unexpected job status")
	assert.Equal(t, "myjob3", out.Jobs[1].Name)
	assert.Equal(t, "unknown", out.Jobs[1].Status, "unexpected job status")
}
