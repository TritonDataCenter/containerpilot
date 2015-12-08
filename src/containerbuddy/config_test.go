package main

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestValidConfigParse(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	var testJson = `{
    "consul": "consul:8500",
    "onStart": "/bin/to/onStart.sh arg1 arg2",
		"preStop": ["/bin/to/preStop.sh","arg1","arg2"],
		"postStop": ["/bin/to/postStop.sh"],
    "services": [
        {
            "name": "serviceA",
            "port": 8080,
            "health": "/bin/to/healthcheck/for/service/A.sh",
            "poll": 30,
            "ttl": 19
        },
        {
            "name": "serviceB",
            "port": 5000,
            "health": "/bin/to/healthcheck/for/service/B.sh",
            "poll": 30,
            "ttl": 103
        }
    ],
    "backends": [
        {
            "name": "upstreamA",
            "poll": 11,
            "onChange": "/bin/to/onChangeEvent/for/upstream/A.sh"
        },
        {
            "name": "upstreamB",
            "poll": 79,
            "onChange": "/bin/to/onChangeEvent/for/upstream/B.sh"
        }
    ]
}
`

	os.Args = []string{"this", "-config", testJson, "/test.sh", "valid1", "--debug"}
	config, _ := loadConfig()

	if len(config.Backends) != 2 || len(config.Services) != 2 {
		t.Errorf("Expected 2 backends and 2 services but got: %v", config)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
	}
	validateCommandParsed(t, "onStart", config.onStartCmd, []string{"/bin/to/onStart.sh", "arg1", "arg2"})
	validateCommandParsed(t, "preStop", config.preStopCmd, []string{"/bin/to/preStop.sh", "arg1", "arg2"})
	validateCommandParsed(t, "postStop", config.postStopCmd, []string{"/bin/to/postStop.sh"})
	validateCommandParsed(t, "health", config.Services[0].healthCheckCmd, []string{"/bin/to/healthcheck/for/service/A.sh"})
	validateCommandParsed(t, "health", config.Services[1].healthCheckCmd, []string{"/bin/to/healthcheck/for/service/B.sh"})
	validateCommandParsed(t, "onChange", config.Backends[0].onChangeCmd, []string{"/bin/to/onChangeEvent/for/upstream/A.sh"})
	validateCommandParsed(t, "onChange", config.Backends[1].onChangeCmd, []string{"/bin/to/onChangeEvent/for/upstream/B.sh"})
}

func TestParseCommandArgs(t *testing.T) {
	if cmd, err := parseCommandArgs(nil); err == nil {
		validateCommandParsed(t, "command", cmd, nil)
	} else {
		t.Errorf("Unexpected parse error: %s", err.Error())
	}

	expected := []string{"/test.sh", "arg1"}
	json1 := json.RawMessage(`"/test.sh arg1"`)
	if cmd, err := parseCommandArgs(json1); err == nil {
		validateCommandParsed(t, "json1", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error json1: %s", err.Error())
	}

	json2 := json.RawMessage(`["/test.sh","arg1"]`)
	if cmd, err := parseCommandArgs(json2); err == nil {
		validateCommandParsed(t, "json2", cmd, expected)
	} else {
		t.Errorf("Unexpected parse error json2: %s", err.Error())
	}

	json3 := json.RawMessage(`{ "a": true }`)
	if _, err := parseCommandArgs(json3); err == nil {
		t.Errorf("Expected parse error for json3")
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

func TestInvalidConfigNoConfigFlag(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "/test.sh", "invalid1", "--debug"}
	if _, err := loadConfig(); err != nil && err.Error() != "-config flag is required." {
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
		"Could not parse configuration: invalid character '<' looking for beginning of value")
}

func TestGetIp(t *testing.T) {
	if ip, _ := getIp([]string{}); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip, _ := getIp(nil); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip, _ := getIp([]string{"eth0"}); ip == "" {
		t.Errorf("Expected to find IP for eth0, but found nothing.")
	}
	if ip, _ := getIp([]string{"eth0", "lo"}); ip == "127.0.0.1" {
		t.Errorf("Expected to find eth0 ip, but found loopback instead")
	}
	if ip, _ := getIp([]string{"lo", "eth0"}); ip != "127.0.0.1" {
		t.Errorf("Expected to find loopback ip, but found: %s", ip)
	}
	if ip, err := getIp([]string{"interface-does-not-exist"}); err == nil {
		t.Errorf("Expected interface not found, but instead got an IP: %s", ip)
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

func testParseExpectError(t *testing.T, testJson string, expected string) {
	os.Args = []string{"this", "-config", testJson, "/test.sh", "test", "--debug"}
	if _, err := loadConfig(); err != nil && err.Error() != expected {
		t.Errorf("Expected %s but got %s", expected, err)
	}
}
