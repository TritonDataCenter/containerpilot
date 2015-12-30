package main

import (
	"encoding/json"
	"flag"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// ------------------------------------------
// Test setup with mock services

type MockAddr struct {
	NetworkAttr string
	StringAttr  string
}

func (addr MockAddr) Network() string {
	return addr.NetworkAttr
}

func (addr MockAddr) String() string {
	return addr.StringAttr
}

// ------------------------------------------

var testJSON = `{
	"consul": "consul:8500",
	"onStart": "/bin/to/onStart.sh arg1 arg2",
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
	os.Args = []string{"this", "-config", testJSON, "/test.sh", "valid1", "--debug"}
	config, _ := loadConfig()
	if !reflect.DeepEqual(config, getConfig()) {
		t.Errorf("Global config was not written after load")
	}

	if len(config.Backends) != 2 || len(config.Services) != 2 {
		t.Errorf("Expected 2 backends and 2 services but got: %v", config)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
	}

	expectedTags := []string{"tag1", "tag2"}
	if !reflect.DeepEqual(config.Services[0].Tags, expectedTags) {
		t.Errorf("Expected tags %s for serviceA, but got: %s", expectedTags, config.Services[0].Tags)
	}

	if config.Backends[0].Tag != "dev" {
		t.Errorf("Expected tag %s for upstreamA, but got: %s", "dev", config.Backends[0].Tag)
	}

	validateCommandParsed(t, "onStart", config.onStartCmd, []string{"/bin/to/onStart.sh", "arg1", "arg2"})
	validateCommandParsed(t, "preStop", config.preStopCmd, []string{"/bin/to/preStop.sh", "arg1", "arg2"})
	validateCommandParsed(t, "postStop", config.postStopCmd, []string{"/bin/to/postStop.sh"})
	validateCommandParsed(t, "health", config.Services[0].healthCheckCmd, []string{"/bin/to/healthcheck/for/service/A.sh"})
	validateCommandParsed(t, "health", config.Services[1].healthCheckCmd, []string{"/bin/to/healthcheck/for/service/B.sh"})
	validateCommandParsed(t, "onChange", config.Backends[0].onChangeCmd, []string{"/bin/to/onChangeEvent/for/upstream/A.sh", "HELLO"})
	validateCommandParsed(t, "onChange", config.Backends[1].onChangeCmd, []string{"/bin/to/onChangeEvent/for/upstream/B.sh"})
}

func TestParseInterfaces(t *testing.T) {
	if interfaces, err := parseInterfaces(nil); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if len(interfaces) > 0 {
		t.Errorf("Expected no interfaces, but got %s", interfaces)
	}

	json1 := json.RawMessage(`"eth0"`)
	expected1 := []string{"eth0"}
	if interfaces, err := parseInterfaces(json1); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if !reflect.DeepEqual(interfaces, expected1) {
		t.Errorf("Expected %s, got: %s", expected1, interfaces)
	}

	json2 := json.RawMessage(`["ethwe","eth0"]`)
	expected2 := []string{"ethwe", "eth0"}
	if interfaces, err := parseInterfaces(json2); err != nil {
		t.Errorf("Unexpected parse error: %s", err.Error())
	} else if !reflect.DeepEqual(interfaces, expected2) {
		t.Errorf("Expected %s, got: %s", expected2, interfaces)
	}

	json3 := json.RawMessage(`{ "a": true }`)
	if _, err := parseInterfaces(json3); err == nil {
		t.Errorf("Expected parse error for json3")
	}
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
	if _, err := loadConfig(); err != nil && err.Error() != "-config flag is required" {
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

func testJSONTemplateParseError(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	testParseExpectError(t,
		`{
    "test": {{ .NO_SUCH_KEY }},
    "test2": "hello"
}`,
		"Parse error at line:col [2:13]")
}

func testJSONTemplateParseError2(t *testing.T) {
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

func TestGetIp(t *testing.T) {
	if ip, _ := getIP([]string{}); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip, _ := getIP(nil); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip, _ := getIP([]string{"eth0"}); ip == "" {
		t.Errorf("Expected to find IP for eth0, but found nothing.")
	}
	if ip, _ := getIP([]string{"eth0", "lo"}); ip == "127.0.0.1" {
		t.Errorf("Expected to find eth0 ip, but found loopback instead")
	}
	if ip, _ := getIP([]string{"lo", "eth0"}); ip != "127.0.0.1" {
		t.Errorf("Expected to find loopback ip, but found: %s", ip)
	}
	if ip, err := getIP([]string{"interface-does-not-exist"}); err == nil {
		t.Errorf("Expected interface not found, but instead got an IP: %s", ip)
	}
}

func TestOnChangeCmd(t *testing.T) {
	cmd1 := strToCmd("/root/examples/test/test.sh doStuff --debug")
	backend := &BackendConfig{
		onChangeCmd: cmd1,
	}
	if _, err := backend.OnChange(); err != nil {
		t.Errorf("Unexpected error OnChange: %s", err)
	}
	// Ensure we can run it more than once
	if _, err := backend.OnChange(); err != nil {
		t.Errorf("Unexpected error OnChange (x2): %s", err)
	}
}

func TestHealthCheck(t *testing.T) {
	cmd1 := strToCmd("/root/examples/test/test.sh doStuff --debug")
	service := &ServiceConfig{
		healthCheckCmd: cmd1,
	}
	if _, err := service.CheckHealth(); err != nil {
		t.Errorf("Unexpected error CheckHealth: %s", err)
	}
	// Ensure we can run it more than once
	if _, err := service.CheckHealth(); err != nil {
		t.Errorf("Unexpected error CheckHealth (x2): %s", err)
	}
}

func TestConfigRequiredFields(t *testing.T) {
	var testConfig *Config

	// --------------
	// Service Tests
	// --------------

	// Missing `name`
	testConfig = unmarshaltestJSON()
	testConfig.Services[0].Name = ""
	validateParseError(t, []string{"`name`"}, testConfig)
	// Missing `poll`
	testConfig = unmarshaltestJSON()
	testConfig.Services[0].Poll = 0
	validateParseError(t, []string{"`poll`", testConfig.Services[0].Name}, testConfig)
	// Missing `ttl`
	testConfig = unmarshaltestJSON()
	testConfig.Services[0].TTL = 0
	validateParseError(t, []string{"`ttl`", testConfig.Services[0].Name}, testConfig)
	// Missing `health`
	testConfig = unmarshaltestJSON()
	testConfig.Services[0].HealthCheckExec = nil
	validateParseError(t, []string{"`health`", testConfig.Services[0].Name}, testConfig)
	// Missing `port`
	testConfig = unmarshaltestJSON()
	testConfig.Services[0].Port = 0
	validateParseError(t, []string{"`port`", testConfig.Services[0].Name}, testConfig)

	// --------------
	// Backend Tests
	// --------------

	// Missing `name`
	testConfig = unmarshaltestJSON()
	testConfig.Backends[0].Name = ""
	validateParseError(t, []string{"`name`"}, testConfig)
	// Missing `poll`
	testConfig = unmarshaltestJSON()
	testConfig.Backends[0].Poll = 0
	validateParseError(t, []string{"`poll`", testConfig.Backends[0].Name}, testConfig)
	// Missing `onChange`
	testConfig = unmarshaltestJSON()
	testConfig.Backends[0].OnChangeExec = nil
	validateParseError(t, []string{"`onChange`", testConfig.Backends[0].Name}, testConfig)
}

func validateParseError(t *testing.T, matchStrings []string, config *Config) {
	if _, err := initializeConfig(config); err == nil {
		t.Errorf("Expected error parsing config")
	} else {
		for _, match := range matchStrings {
			if !strings.Contains(err.Error(), match) {
				t.Errorf("Expected message does not contain %s: %s", match, err)
			}
		}
	}
}

func TestInterfaceIpsLoopback(t *testing.T) {
	interfaces := make([]net.Interface, 1)

	interfaces[0] = net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback,
	}

	interfaceIps, err := getinterfaceIPs(interfaces)

	if err != nil {
		t.Error(err)
		return
	}

	/* Because we are testing inside of Docker we can expect that the loopback
	 * interface to always be on the IPv4 address 127.0.0.1 and to be at
	 * index 1 */

	if len(interfaceIps) != 1 {
		t.Error("No IPs were parsed from interface. Expecting: 127.0.0.1")
	}

	if interfaceIps[0].IP != "127.0.0.1" {
		t.Error("Expecting loopback interface [127.0.0.1] to be returned")
	}
}

func TestInterfaceIpsError(t *testing.T) {
	interfaces := make([]net.Interface, 2)

	interfaces[0] = net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  "lo",
		Flags: net.FlagUp | net.FlagLoopback,
	}
	interfaces[1] = net.Interface{
		Index:        -1,
		MTU:          65536,
		Name:         "barf",
		Flags:        net.FlagUp | net.FlagBroadcast | net.FlagMulticast,
		HardwareAddr: []byte{0x10, 0xC3, 0x7B, 0x45, 0xA2, 0xFF},
	}

	interfaceIps, err := getinterfaceIPs(interfaces)

	if err != nil {
		t.Error(err)
		return
	}

	/* We expect to get only a single valid ip address back because the second
	 * value is junk. */

	if len(interfaceIps) != 1 {
		t.Error("No IPs were parsed from interface. Expecting: 127.0.0.1")
	}

	if interfaceIps[0].IP != "127.0.0.1" {
		t.Error("Expecting loopback interface [127.0.0.1] to be returned")
	}
}

func TestParseIPv4FromSingleAddress(t *testing.T) {
	expectedIP := "192.168.22.123"

	intf := net.Interface{
		Index:        -1,
		MTU:          1500,
		Name:         "fake",
		Flags:        net.FlagUp | net.FlagBroadcast | net.FlagMulticast,
		HardwareAddr: []byte{0x10, 0xC3, 0x7B, 0x45, 0xA2, 0xFF},
	}

	addr := MockAddr{
		NetworkAttr: "ip+net",
		StringAttr:  expectedIP + "/8",
	}

	ifaceIP, err := parseIPFromAddress(addr, intf)

	if err != nil {
		t.Error(err)
		return
	}

	if ifaceIP.IP != expectedIP {
		t.Errorf("IP didn't match expectation. Actual: %s Expected: %s",
			ifaceIP.IP, expectedIP)
	}
}

func TestParseIPv4FromIPv6AndIPv4AddressesIPv4First(t *testing.T) {
	expectedIP := "192.168.22.123"

	intf := net.Interface{
		Index:        -1,
		MTU:          1500,
		Name:         "fake",
		Flags:        net.FlagUp | net.FlagBroadcast | net.FlagMulticast,
		HardwareAddr: []byte{0x10, 0xC3, 0x7B, 0x45, 0xA2, 0xFF},
	}

	addr := MockAddr{
		NetworkAttr: "ip+net",
		StringAttr:  expectedIP + "/8" + " fe80::12c3:7bff:fe45:a2ff/64",
	}

	ifaceIP, err := parseIPFromAddress(addr, intf)

	if err != nil {
		t.Error(err)
		return
	}

	if ifaceIP.IP != expectedIP {
		t.Errorf("IP didn't match expectation. Actual: %s Expected: %s",
			ifaceIP.IP, expectedIP)
	}
}

func TestParseIPv4FromIPv6AndIPv4AddressesIPv6First(t *testing.T) {
	expectedIP := "192.168.22.123"

	intf := net.Interface{
		Index:        -1,
		MTU:          1500,
		Name:         "fake",
		Flags:        net.FlagUp | net.FlagBroadcast | net.FlagMulticast,
		HardwareAddr: []byte{0x10, 0xC3, 0x7B, 0x45, 0xA2, 0xFF},
	}

	addr := MockAddr{
		NetworkAttr: "ip+net",
		StringAttr:  "fe80::12c3:7bff:fe45:a2ff/64 " + expectedIP + "/8",
	}

	ifaceIP, err := parseIPFromAddress(addr, intf)

	if err != nil {
		t.Error(err)
		return
	}

	if ifaceIP.IP != expectedIP {
		t.Errorf("IP didn't match expectation. Actual: %s Expected: %s",
			ifaceIP.IP, expectedIP)
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
	os.Args = []string{"this", "-config", testJSON, "/test.sh", "test", "--debug"}
	if _, err := loadConfig(); err != nil && !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected %s but got %s", expected, err)
	}
}

func unmarshaltestJSON() *Config {
	config, _ := unmarshalConfig([]byte(testJSON))
	return config
}
