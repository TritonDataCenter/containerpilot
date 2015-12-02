package main

import (
	"flag"
	"os"
	"testing"
)

func TestValidConfigParse(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	var testJson = `{
    "consul": "consul:8500",
    "onStart": "/bin/to/onStart.sh",
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
	if config.OnStart != "/bin/to/onStart.sh" {
		t.Errorf("onStart not configured")
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
	if ip,_ := getIp(""); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip,_ := getIp("eth0"); ip == "" {
		t.Errorf("Expected to find IP for eth0, but found nothing.")
	}
	if ip,_ := getIp("lo"); ip != "127.0.0.1" {
		t.Errorf("Expected to find loopback ip, but found: %s",ip)
	}
	if ip,err := getIp("interface-does-not-exist"); err == nil {
		t.Errorf("Expected interface not found, but instead got an IP: %s",ip)
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
