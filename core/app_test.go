package core

import (
	"flag"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/config"
)

// ------------------------------------------

var testJSON = `{
	"consul": "consul:8500",
	"preStart": "/bin/to/preStart.sh arg1 arg2",
	"preStop": ["/bin/to/preStop.sh","arg1","arg2"],
	"postStop": ["/bin/to/postStop.sh"],
	"services": [
			{
					"name": "serviceA",
					"port": 8080,
					"interfaces": "eth0",
					"health": "/bin/to/healthcheck/for/service/A.sh",
					"poll": 30,
					"ttl": 19,
					"tags": ["tag1","tag2"]
			},
			{
					"name": "serviceB",
					"port": 5000,
					"interfaces": ["ethwe","eth0"],
					"health": "/bin/to/healthcheck/for/service/B.sh",
					"poll": 30,
					"ttl": 103
			}
	],
	"backends": [
			{
					"name": "upstreamA",
					"poll": 11,
					"onChange": "/bin/to/onChangeEvent/for/upstream/A.sh {{.TEST}}",
					"tag": "dev"
			},
			{
					"name": "upstreamB",
					"poll": 79,
					"onChange": "/bin/to/onChangeEvent/for/upstream/B.sh {{.ENV_NOT_FOUND}}"
			}
	]
}
`

func TestValidConfigParse(t *testing.T) {
	defer argTestCleanup(argTestSetup())

	os.Setenv("TEST", "HELLO")
	os.Args = []string{"this", "-config", testJSON, "/testdata/test.sh", "valid1", "--debug"}
	app, _ := LoadApp()

	if len(app.Backends) != 2 || len(app.Services) != 2 {
		t.Errorf("Expected 2 backends and 2 services but got: %v", app)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/testdata/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
	}

	expectedTags := []string{"tag1", "tag2"}
	if !reflect.DeepEqual(app.Services[0].Tags, expectedTags) {
		t.Errorf("Expected tags %s for serviceA, but got: %s", expectedTags, app.Services[0].Tags)
	}

	if app.Services[1].Tags != nil {
		t.Errorf("Expected no tags for serviceB, but got: %s", app.Services[1].Tags)
	}

	if app.Backends[0].Tag != "dev" {
		t.Errorf("Expected tag %s for upstreamA, but got: %s", "dev", app.Backends[0].Tag)
	}

	if app.Backends[1].Tag != "" {
		t.Errorf("Expected no tag for upstreamB, but got: %s", app.Backends[1].Tag)
	}

	validateCommandParsed(t, "preStart", app.PreStartCmd, []string{"/bin/to/preStart.sh", "arg1", "arg2"})
	validateCommandParsed(t, "preStop", app.PreStopCmd, []string{"/bin/to/preStop.sh", "arg1", "arg2"})
	validateCommandParsed(t, "postStop", app.PostStopCmd, []string{"/bin/to/postStop.sh"})
}

func TestServiceConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testJSON = []byte(`{"consul": "consul:8500", "services": [
                           {"name": "", "port": 8080, "poll": 30, "ttl": 19 }]}`)
	validateParseError(t, testJSON, []string{"`name`"})

	// Missing `poll`
	testJSON = []byte(`{"consul": "consul:8500", "services": [
                       {"name": "name", "port": 8080, "ttl": 19}]}`)
	validateParseError(t, testJSON, []string{"`poll`"})

	// Missing `ttl`
	testJSON = []byte(`{"consul": "consul:8500", "services": [
                       {"name": "name", "port": 8080, "poll": 19}]}`)
	validateParseError(t, testJSON, []string{"`ttl`"})

	testJSON = []byte(`{"consul": "consul:8500", "services": [
                       {"name": "name", "poll": 19, "ttl": 19}]}`)
	validateParseError(t, testJSON, []string{"`port`"})
}

func TestBackendConfigRequiredFields(t *testing.T) {
	// Missing `name`
	var testJSON = []byte(`{"consul": "consul:8500", "backends": [
                           {"name": "", "poll": 30, "ttl": 19, "onChange": "true"}]}`)
	validateParseError(t, testJSON, []string{"`name`"})

	// Missing `poll`
	testJSON = []byte(`{"consul": "consul:8500", "backends": [
                       {"name": "name", "ttl": 19, "onChange": "true"}]}`)
	validateParseError(t, testJSON, []string{"`poll`"})

	// Missing `onChange`
	testJSON = []byte(`{"consul": "consul:8500", "backends": [
                       {"name": "name", "poll": 19, "ttl": 19 }]}`)
	validateParseError(t, testJSON, []string{"`onChange`"})
}

func TestInvalidConfigNoConfigFlag(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "/testdata/test.sh", "invalid1", "--debug"}
	if _, err := LoadApp(); err != nil && err.Error() != "-config flag is required" {
		t.Errorf("Expected error but got %s", err)
	}
}

func TestInvalidConfigParseNoDiscovery(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t, "{}", "No discovery backend defined")
}

func TestInvalidConfigParseFile(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t, "file:///xxxx",
		"Could not read config file: open /xxxx: no such file or directory")
}

func TestInvalidConfigParseNotJson(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t, "<>",
		"Parse error at line:col [1:1]")
}

func TestJSONTemplateParseError(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t,
		`{
    "test": {{ .NO_SUCH_KEY }},
    "test2": "hello"
}`,
		"Parse error at line:col [2:13]")
}

func TestJSONTemplateParseError2(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t,
		`{
    "test1": "1",
    "test2": 2,
    "test3": false,
    test2: "hello"
}`,
		"Parse error at line:col [5:5]")
}

func TestParseTrailingComma(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t,
		`{
			"consul": "consul:8500",
			"tasks": [{
				"command": ["echo","hi"]
			},
		]
	}`, "Do you have an extra comma somewhere?")
}

func TestMetricServiceCreation(t *testing.T) {

	jsonFragment := `{
    "consul": "consul:8500",
    "telemetry": {
      "port": 9090
    }
  }`
	config, err := config.ParseConfig(jsonFragment)
	if err != nil {
		t.Fatalf("Error parsing config: %v", err)
	}
	app, err := NewApp(config)
	if err != nil {
		t.Fatalf("Got unexpected error deserializing JSON config: %v", err)
	}
	if _, err := NewApp(config); err != nil {
		t.Fatalf("Got error while initializing config: %v", err)
	} else {
		if len(app.Services) != 1 {
			t.Errorf("Expected telemetry service but got %v", app.Services)
		} else {
			service := app.Services[0]
			if service.Name != "containerpilot" {
				t.Errorf("Got incorrect service back: %v", service)
			}
		}
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
		t.Errorf("Expected %s but got %s", expected, err)
	}
}

func validateParseError(t *testing.T, input []byte, matchStrings []string) {
	if cfg, err := config.UnmarshalConfig([]byte(input)); err != nil {
		t.Errorf("Unexpected error parsing config: %v", err)
	} else {
		if _, err := NewApp(cfg); err == nil {
			t.Errorf("Expected error parsing config")
		} else {
			for _, match := range matchStrings {
				if !strings.Contains(err.Error(), match) {
					t.Errorf("Expected message does not contain %s: %s", match, err)
				}
			}
		}
	}
}

func validateCommandParsed(t *testing.T, name string, parsed *exec.Cmd, expected []string) {
	if expected == nil {
		if parsed != nil {
			t.Errorf("%s has Cmd, but expected nil", name)
		}
		return
	}
	if parsed == nil {
		t.Errorf("%s not configured", name)
	}
	if parsed.Path != expected[0] {
		t.Errorf("%s path not configured: %s != %s", name, parsed.Path, expected[0])
	}
	if !reflect.DeepEqual(parsed.Args, expected) {
		t.Errorf("%s arguments not configured: %s != %s", name, parsed.Args, expected)
	}
}
