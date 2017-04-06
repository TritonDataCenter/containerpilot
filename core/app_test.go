package core

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/commands"
	_ "github.com/joyent/containerpilot/discovery/consul"
)

/*
TODO v3: a LOT of the these tests should be moved to the config package
*/

func TestServiceConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testJSON = `{"consul": "consul:8500", "services": [
                           {"name": "", "port": 8080, "poll": 30, "ttl": 19 }]}`
	validateParseError(t, testJSON, []string{"`name`"})

	// Missing `poll`
	testJSON = `{"consul": "consul:8500", "services": [
                       {"name": "name", "port": 8080, "ttl": 19}]}`
	validateParseError(t, testJSON, []string{"`poll`"})

	// Missing `ttl`
	testJSON = `{"consul": "consul:8500", "services": [
                       {"name": "name", "port": 8080, "poll": 19}]}`
	validateParseError(t, testJSON, []string{"`ttl`"})

	testJSON = `{"consul": "consul:8500", "services": [
                       {"name": "name", "poll": 19, "ttl": 19}]}`
	validateParseError(t, testJSON, []string{"`port`"})
}

func TestBackendConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testJSON = `{"consul": "consul:8500", "backends": [
                           {"name": "", "poll": 30, "onChange": "true"}]}`
	validateParseError(t, testJSON, []string{"`name`"})

	// Missing `poll`
	testJSON = `{"consul": "consul:8500", "backends": [
                       {"name": "name", "onChange": "true"}]}`
	validateParseError(t, testJSON, []string{"`poll`"})

	// Missing `onChange`
	testJSON = `{"consul": "consul:8500", "backends": [
                       {"name": "name", "poll": 19 }]}`
	validateParseError(t, testJSON, []string{"`onChange`"})
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
	testParseExpectError(t, "{}", "no discovery backend defined")
}

func TestInvalidConfigParseFile(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t, "file:///xxxx",
		"could not read config file: open /xxxx: no such file or directory")
}

func TestInvalidConfigParseNotJson(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t, "<>",
		"parse error at line:col [1:1]")
}

func TestJSONTemplateParseError(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t,
		`{
    "test": {{ .NO_SUCH_KEY }},
    "test2": "hello"
}`,
		"parse error at line:col [2:13]")
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

func testParseExpectError(t *testing.T, testJSON string, expected string) {
	os.Args = []string{"this", "-config", testJSON, "/testdata/test.sh", "test", "--debug"}
	if _, err := LoadApp(); err != nil && !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %s but got %s", expected, err)
	}
}

func validateParseError(t *testing.T, testJSON string, matchStrings []string) {
	if _, err := NewApp(testJSON); err == nil {
		t.Errorf("expected error parsing config")
	} else {
		for _, match := range matchStrings {
			if !strings.Contains(err.Error(), match) {
				t.Errorf("expected message does not contain %s: %s", match, err)
			}
		}
	}
}

func validateCommandParsed(t *testing.T, name string, parsed *commands.Command,
	expectedExec string, expectedArgs []string) {
	if parsed == nil {
		t.Errorf("%s not configured", name)
	}
	if !reflect.DeepEqual(parsed.Exec, expectedExec) {
		t.Errorf("%s executable not configured: %s != %s", name, parsed.Exec, expectedExec)
	}
	if !reflect.DeepEqual(parsed.Args, expectedArgs) {
		t.Errorf("%s arguments not configured: %s != %s", name, parsed.Args, expectedArgs)
	}
}
