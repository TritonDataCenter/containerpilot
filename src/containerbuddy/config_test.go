package main

import (
	"flag"
	"net"
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
	if ip := getIp(false,""); ip == "" {
		t.Errorf("Expected private IP but got nothing")
	}
	if ip := getIp(true,""); ip != "" {
		t.Errorf("Expected no public IP but got: %s", ip)
	}
	if ip := getIp(true,"eth0"); ip == "" {
		t.Errorf("Expected interface IP but got nothing")
	}
}

func TestIpCheckPrivate(t *testing.T) {
	var privateIps = []string{
		"192.168.1.117",
		"172.17.1.1",
		"10.1.1.13",
	}
	for _, ipAddr := range privateIps {
		ip := net.ParseIP(ipAddr)
		if isPublicIp(ip) {
			t.Errorf("Expected %s to be identified as private IP but got public.", ip)
		}
	}
}

func TestIpCheckPublic(t *testing.T) {
	var publicIps = []string{
		"8.8.8.8",
		"72.2.117.118",
	}
	for _, ipAddr := range publicIps {
		ip := net.ParseIP(ipAddr)
		if !isPublicIp(ip) {
			t.Errorf("Expected %s to be identified as public IP but got private.", ip)
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

func testParseExpectError(t *testing.T, testJson string, expected string) {
	os.Args = []string{"this", "-config", testJson, "/test.sh", "test", "--debug"}
	if _, err := loadConfig(); err != nil && err.Error() != expected {
		t.Errorf("Expected %s but got %s", expected, err)
	}
}
