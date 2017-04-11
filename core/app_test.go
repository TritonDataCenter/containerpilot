package core

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/joyent/containerpilot/discovery/consul"
	"github.com/joyent/containerpilot/tests/assert"
)

/*
TODO v3: a LOT of the these tests should be moved to the config package
*/

func TestJobConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testCfg = `{"consul": "consul:8500", jobs: [
                    {"name": "", "port": 8080, health: {interval: 30, "ttl": 19 }}]}`
	_, err := NewApp(testCfg)
	assert.Error(t, err, "unable to parse jobs: 'name' must not be blank")

	// Missing `interval`
	testCfg = `{"consul": "consul:8500", jobs: [
                {"name": "name", "port": 8080, health: {ttl: 19}}]}`
	_, err = NewApp(testCfg)
	assert.Error(t, err, "unable to parse jobs: job[name].health.interval must be > 0")

	// Missing `ttl`
	testCfg = `{"consul": "consul:8500", jobs: [
                {"name": "name", "port": 8080, health: {interval: 19}}]}`
	_, err = NewApp(testCfg)
	assert.Error(t, err, "unable to parse jobs: job[name].health.ttl must be > 0")
}

func TestBackendConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testCfg = `{"consul": "consul:8500", watches: [{"name": "", "interval": 30}]}`
	_, err := NewApp(testCfg)
	assert.Error(t, err, "unable to parse watches: 'name' must not be blank")

	// Missing `interval`
	testCfg = `{"consul": "consul:8500", watches: [{"name": "name"}]}`
	_, err = NewApp(testCfg)
	assert.Error(t, err, "unable to parse watches: watch[name].interval must be > 0")
}

func TestInvalidConfigNoConfigFlag(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "/testdata/test.sh", "invalid1", "--debug"}
	if _, err := LoadApp(); err != nil && err.Error() != "-config flag is required" {
		t.Errorf("expected error but got %s", err)
	}
}

func TestInvalidConfigParseNoDiscovery(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "{}"}
	_, err := LoadApp()
	assert.Error(t, err, "no discovery backend defined")
}

func TestInvalidConfigMissingFile(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "file:///xxxx"}
	_, err := LoadApp()
	assert.Error(t, err,
		"could not read config file: open /xxxx: no such file or directory")
}

func TestInvalidConfigParseNotJson(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "<>"}
	_, err := LoadApp()
	assert.Error(t, fmt.Errorf("%s", err.Error()[:29]),
		"parse error at line:col [1:1]")
}

func TestInvalidConfigParseTemplateError(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	// this config is missing quotes around the template
	badCfg := `{"test": {{ .NO_SUCH_KEY }}, "test2": "hello"}`
	os.Args = []string{"this", "-config", badCfg}
	_, err := LoadApp()
	assert.Error(t, fmt.Errorf("%s", err.Error()[:30]),
		"parse error at line:col [1:10]")
}

func TestRenderArgs(t *testing.T) {
	flags := []string{"-name", "{{ .HOSTNAME }}"}
	expected := os.Getenv("HOSTNAME")
	if expected == "" {
		// not all environments use this variable as a hostname but
		// we really just want to make sure it's being rendered
		expected, _ = os.Hostname()
		os.Setenv("HOSTNAME", expected)
	}
	if got := getArgs(flags)[1]; got != expected {
		t.Errorf("expected %v but got %v for rendered hostname", expected, got)
	}

	// invalid template should just be returned unchanged
	flags = []string{"-name", "{{ .HOSTNAME }"}
	expected = "{{ .HOSTNAME }"
	if got := getArgs(flags)[1]; got != expected {
		t.Errorf("expected %v but got %v for unrendered hostname", expected, got)
	}
}

func TestControlServerCreation(t *testing.T) {

	jsonFragment := `{
    "consul": "consul:8500"
  }`

	app, err := NewApp(jsonFragment)
	if err != nil {
		t.Fatalf("got error while initializing config: %v", err)
	}

	if app.ControlServer == nil {
		t.Error("expected control server to not be nil")
	}
}

func TestMetricServiceCreation(t *testing.T) {

	jsonFragment := `{
    "consul": "consul:8500",
    "telemetry": {
      "interfaces": ["inet"],
      "port": 9090
    }
  }`
	if app, err := NewApp(jsonFragment); err != nil {
		t.Fatalf("got error while initializing config: %v", err)
	} else {
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
}

func TestPidEnvVar(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "{}", "/testdata/test.sh"}
	if _, err := LoadApp(); err == nil {
		t.Fatalf("expected error in LoadApp but got none")
	}
	if pid := os.Getenv("CONTAINERPILOT_PID"); pid == "" {
		t.Errorf("expected CONTAINERPILOT_PID to be set even on error")
	}
}

// ----------------------------------------------------
// test helpers

func argTestSetup() []string {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = nil
	return os.Args
}

func argTestCleanup(oldArgs []string) {
	os.Args = oldArgs
}
