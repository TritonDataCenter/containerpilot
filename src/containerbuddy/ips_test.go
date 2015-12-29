package main

import (
	"encoding/json"
	"math/rand"
	"net"
	"reflect"
	"sort"
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

func TestGetIp(t *testing.T) {
	if ip, _ := GetIP([]string{}); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip, _ := GetIP(nil); ip == "" {
		t.Errorf("Expected default interface to yield an IP, but got nothing.")
	}
	if ip, _ := GetIP([]string{"eth0"}); ip == "" {
		t.Errorf("Expected to find IP for eth0, but found nothing.")
	}
	if ip, _ := GetIP([]string{"eth0", "lo"}); ip == "127.0.0.1" {
		t.Errorf("Expected to find eth0 ip, but found loopback instead")
	}
	if ip, _ := GetIP([]string{"lo", "eth0"}); ip != "127.0.0.1" {
		t.Errorf("Expected to find loopback ip, but found: %s", ip)
	}
	if ip, err := GetIP([]string{"interface-does-not-exist"}); err == nil {
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

	for _, ip := range interfaceIps {
		if ip.Name != "lo" {
			t.Errorf("Expecting loopback interface, but got: %s", ip.Name)
		}
		if ip.IsIPv4() && ip.IPString() != "127.0.0.1" {
			t.Errorf("Expecting loopback interface [127.0.0.1] to be returned, got %s", ip.IPString())
		}
		if !ip.IsIPv4() && ip.IPString() != "::1" {
			t.Errorf("Expecting loopback interface [::1] to be returned, got %s", ip.IPString())
		}
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

	if len(interfaceIps) == 0 {
		t.Error("No IPs were parsed from interface. Expecting: 127.0.0.1")
	}

	for _, ip := range interfaceIps {
		if ip.Name != "lo" {
			t.Errorf("Expecting loopback interface, but got: %s", ip.Name)
		}
		if ip.IsIPv4() && ip.IPString() != "127.0.0.1" {
			t.Errorf("Expecting loopback interface [127.0.0.1] to be returned, got %s", ip.IPString())
		}
		if !ip.IsIPv4() && ip.IPString() != "::1" {
			t.Errorf("Expecting loopback interface [::1] to be returned, got %s", ip.IPString())
		}
	}
}

func TestInterfaceSpecParse(t *testing.T) {
	// Test Error Cases
	testSpecError(t, "")           // Nothing
	testSpecError(t, "!")          // Nonsense
	testSpecError(t, "127.0.0.1")  // No Network
	testSpecError(t, "eth0:inet5") // Invalid IP Version
	testSpecError(t, "eth0[0]")    // Invalid Index

	// Test Interface Case
	testSpecInterfaceName(t, "eth0", "eth0", false, 0)
	testSpecInterfaceName(t, "eth0:inet6", "eth0", true, 0)
	testSpecInterfaceName(t, "eth0[1]", "eth0", false, 1)
	testSpecInterfaceName(t, "eth0[2]", "eth0", false, 2)
	testSpecInterfaceName(t, "inet", "*", false, 0)
	testSpecInterfaceName(t, "inet6", "*", true, 0)

	// Test CIDR Case
	testSpecCIDR(t, "10.0.0.0/16")
	testSpecCIDR(t, "fdc6:238c:c4bc::/48")
}

func testSpecError(t *testing.T, specStr string) {
	if spec, err := parseInterfaceSpec(specStr); err == nil {
		t.Errorf("Expected error but got %s", spec)
	}
}

func testSpecInterfaceName(t *testing.T, specStr string, name string, ipv6 bool, index int) {
	spec, err := parseInterfaceSpec(specStr)
	if err != nil {
		t.Errorf("Expected parse to succeed, but got error: %s", err)
	}
	if spec.Name != name {
		t.Errorf("Expected to parse interface name %s but got %s", name, spec.Name)
	}
	if spec.IPv6 != ipv6 {
		if ipv6 {
			t.Errorf("Expected spec %s to be IPv6", spec)
		} else {
			t.Errorf("Expected spec %s to be IPv4", spec)
		}
	}
	if spec.Index != index {
		t.Errorf("Expected index to be %d but was %d", index, spec.Index)
	}
}

func testSpecCIDR(t *testing.T, specStr string) {
	spec, err := parseInterfaceSpec(specStr)
	if err != nil {
		t.Errorf("Expected parse to succeed, but got error: %s", err)
	}
	if spec.Network == nil {
		t.Errorf("Expected spec to be a network CIDR")
	}
}

func TestFindIPWithSpecs(t *testing.T) {
	iips := getTestIPs()

	// Loopback
	testIPSpec(t, iips, "127.0.0.1", "lo")
	testIPSpec(t, iips, "::1", "lo:inet6")

	// Interface Name
	testIPSpec(t, iips, "10.2.0.1", "eth0")
	testIPSpec(t, iips, "10.2.0.1", "eth0:inet")
	testIPSpec(t, iips, "", "eth0:inet6")

	// Indexes
	testIPSpec(t, iips, "192.168.1.100", "eth0[2]")
	testIPSpec(t, iips, "", "eth0[3]")

	// IPv4 CIDR
	testIPSpec(t, iips, "10.0.0.100", "10.0.0.0/16")
	testIPSpec(t, iips, "10.1.0.200", "10.1.0.0/16")
	testIPSpec(t, iips, "10.2.0.1", "10.0.0.0/8")

	// IPv6 CIDR
	testIPSpec(t, iips, "fdc6:238c:c4bc::1", "eth2:inet6")

	// First IPv4
	testIPSpec(t, iips, "10.2.0.1", "inet")

	// First IPv6
	testIPSpec(t, iips, "fdc6:238c:c4bc::1", "inet6")

	// Test Multiple
	testIPSpec(t, iips, "fdc6:238c:c4bc::1", "eth3", "fdc6:238c:c4bc::/48", "inet", "inet6")
	testIPSpec(t, iips, "10.0.0.100", "eth3", "10.0.0.0/16", "inet", "inet6", "fdc6:238c:c4bc::/48")

	// Test that inet and inet6 will never find the loopback address
	loopback := []interfaceIP{
		newInterfaceIP("lo", "::1"),
		newInterfaceIP("lo", "127.0.0.1"),
	}
	testIPSpec(t, loopback, "", "inet")
	testIPSpec(t, loopback, "", "inet6")
}

func testIPSpec(t *testing.T, iips []interfaceIP, expectedIP string, specList ...string) {
	specs, err := parseInterfaceSpecs(specList)
	if err != nil {
		t.Fatalf("Fatal parse error of spec list: %s, %s", specList, err)
	}
	foundIP, err := findIPWithSpecs(specs, iips)
	if err != nil && expectedIP != "" {
		t.Errorf("Expected to find an IP, but got an error instead: %s", err)
	}
	if foundIP != expectedIP {
		t.Errorf("Expected to find IP %s but found %s instead", expectedIP, foundIP)
	}
}

func TestInterfaceIPSorting(t *testing.T) {
	sortedIIPs := getTestIPs()
	var unsortedIIPs = make([]interfaceIP, len(sortedIIPs))
	copy(unsortedIIPs, sortedIIPs)

	// Shuffle unsortedIIPs
	rand.Seed(1) // Deterministic
	for i := range unsortedIIPs {
		j := rand.Intn(i + 1)
		unsortedIIPs[i], unsortedIIPs[j] = unsortedIIPs[j], unsortedIIPs[i]
	}

	// Do Stable Sort
	sort.Stable(ByInterfaceThenIP(unsortedIIPs))

	if !reflect.DeepEqual(unsortedIIPs, sortedIIPs) {
		t.Errorf("Interface IPs are not sorted as expected")

		t.Log("=== EXPECTED ===")
		for _, ip := range sortedIIPs {
			t.Logf("%s: %s", ip.Name, ip)
		}

		t.Log("=== ACTUAL ===")
		for _, ip := range unsortedIIPs {
			t.Logf("%s: %s", ip.Name, ip)
		}
	}
}

// -------- Helper Functions

func getTestIPs() []interfaceIP {
	return []interfaceIP{
		newInterfaceIP("eth0", "10.2.0.1"),
		newInterfaceIP("eth0", "192.168.1.100"),
		newInterfaceIP("eth1", "10.0.0.100"),
		newInterfaceIP("eth1", "10.0.0.200"),
		newInterfaceIP("eth2", "10.1.0.200"),
		newInterfaceIP("eth2", "fdc6:238c:c4bc::1"),
		newInterfaceIP("lo", "::1"),
		newInterfaceIP("lo", "127.0.0.1"),
	}
}

func newInterfaceIP(name string, ip string) interfaceIP {
	return interfaceIP{
		Name: name,
		IP:   net.ParseIP(ip),
	}
}
