package core

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tritondatacenter/containerpilot/discovery"
	"github.com/tritondatacenter/containerpilot/events"
	"github.com/tritondatacenter/containerpilot/jobs"
	"github.com/tritondatacenter/containerpilot/tests/mocks"
)

/*
Many of these tests are effectively smoke tests for making sure
the core and config packages are working together
*/

func TestJobConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testCfg = `{"consul": "consul:8500", jobs: [
					{"name": "", "port": 8080, health: {interval: 30, "ttl": 19 }}]}`
	f1 := testCfgToTempFile(t, testCfg)
	defer os.Remove(f1.Name())
	_, err := NewApp(f1.Name())
	assert.Error(t, err, "unable to parse jobs: 'name' must not be blank")

	// Missing `interval`
	testCfg = `{"consul": "consul:8500", jobs: [
				{"name": "name", "port": 8080, health: {ttl: 19}}]}`
	f2 := testCfgToTempFile(t, testCfg)
	defer os.Remove(f2.Name())
	_, err = NewApp(f2.Name())
	assert.Error(t, err, "unable to parse jobs: job[name].health.interval must be > 0")

	// Missing `ttl`
	testCfg = `{"consul": "consul:8500", jobs: [
				{"name": "name", "port": 8080, health: {interval: 19}}]}`
	f3 := testCfgToTempFile(t, testCfg)
	defer os.Remove(f3.Name())
	_, err = NewApp(f3.Name())
	assert.Error(t, err, "unable to parse jobs: job[name].health.ttl must be > 0")
}

func TestWatchConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testCfg = `{"consul": "consul:8500", watches: [{"name": "", "interval": 30}]}`
	f1 := testCfgToTempFile(t, testCfg)
	defer os.Remove(f1.Name())
	_, err := NewApp(f1.Name())
	assert.Error(t, err, "unable to parse watches: 'name' must not be blank")

	// Missing `interval`
	testCfg = `{"consul": "consul:8500", watches: [{"name": "name"}]}`
	f2 := testCfgToTempFile(t, testCfg)
	defer os.Remove(f2.Name())
	_, err = NewApp(f2.Name())
	assert.Error(t, err, "unable to parse watches: watch[name].interval must be > 0")
}

func TestMetricServiceCreation(t *testing.T) {

	f := testCfgToTempFile(t, `{
	"consul": "consul:8500",
	"telemetry": {
	  "interfaces": ["inet", "lo0"],
	  "port": 9090
	}
  }`)
	defer os.Remove(f.Name())
	app, err := NewApp(f.Name())
	if err != nil {
		t.Fatalf("got error while initializing config: %v", err)
	}
	if len(app.Jobs) != 1 {
		for _, job := range app.Jobs {
			fmt.Printf("%+v\n", job.Name)
		}
		t.Errorf("expected telemetry service but got %v", app.Jobs)
	} else {
		service := app.Jobs[0]
		if service.Name != "containerpilot" {
			t.Errorf("got incorrect service back: %v", service)
		}
		for _, envVar := range os.Environ() {
			if strings.HasPrefix(envVar, "CONTAINERPILOT_CONTAINERPILOT_IP") {
				return
			}
		}
		t.Errorf("did not find CONTAINERPILOT_CONTAINERPILOT_IP env var")
	}
}

// Test configuration reload
func TestReloadConfig(t *testing.T) {
	cfg := &jobs.Config{
		Name:       "test-service",
		Port:       1,
		Interfaces: []string{"inet"},
		Exec:       []string{"./testdata/test.sh", "interruptSleep"},
		Health: &jobs.HealthConfig{
			Heartbeat: 1,
			TTL:       1,
		},
	}
	cfg.Validate(&mocks.NoopDiscoveryBackend{})
	job := jobs.NewJob(cfg)
	app := EmptyApp()
	app.StopTimeout = 5
	app.Jobs = []*jobs.Job{job}
	app.Bus = events.NewEventBus()

	// write invalid config to temp file and assign it as app config
	f := testCfgToTempFile(t, `invalid`)
	defer os.Remove(f.Name())
	app.ConfigFlag = f.Name()

	err := app.reload()
	if err == nil {
		t.Errorf("invalid configuration did not return error")
	}

	// write new valid configuration
	validConfig := []byte(`{ "consul": "newconsul:8500" }`)
	f2, err := os.Create(f.Name()) // we'll just blow away the old file
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f2.Write(validConfig); err != nil {
		t.Fatal(err)
	}
	if err := f2.Close(); err != nil {
		t.Fatal(err)
	}
	err = app.reload()
	if err != nil {
		t.Errorf("valid configuration returned error: %v", err)
	}
	discSvc := app.Discovery
	if svc, ok := discSvc.(*discovery.Consul); !ok || svc == nil {
		t.Errorf("configuration was not reloaded: %v", discSvc)
	}
}

// ----------------------------------------------------
// test helpers

// write the configuration to a tempfile. caller is responsible
// for calling 'defer os.Remove(f.Name())' when done
func testCfgToTempFile(t *testing.T, text string) *os.File {
	f, err := os.CreateTemp(".", "test-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(text)); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return f
}
