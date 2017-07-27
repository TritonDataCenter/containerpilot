package services

import (
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

var (
	lo  = getLocalhostIfaceName()
	lo6 = fmt.Sprintf("%s:inet6", lo)
)

func getLocalhostIfaceName() string {
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "lo") {
			return iface.Name
		}
	}
	return ""
}

func TestGetIp(t *testing.T) {

	ip, _ := GetIP([]string{lo, "inet"})
	assert.Equal(t, ip, "127.0.0.1", "expected to find loopback IP")

	ip, err := GetIP([]string{"interface-does-not-exist"})
	assert.Error(t, err, "expected interface not found, but instead got an IP")

	ip, _ = GetIP([]string{"static:192.168.1.100", lo})
	assert.Equal(t, ip, "192.168.1.100", "expected to find static IP")

	// these tests can't pass if the test runner doesn't have a valid inet
	// address, so we'll skip these tests in that environment.
	interfaces, _ := net.Interfaces()
	allIps, _ := getinterfaceIPs(interfaces)
	for _, ip := range allIps {
		if ip.IsIPv4() && !ip.IP.IsLoopback() {
			if ip, err := GetIP([]string{}); ip == "" {
				t.Errorf("expected default interface to yield an IP, but got nothing: %v", err)
			}
			if ip, _ := GetIP(nil); ip == "" {
				t.Errorf("expected default interface to yield an IP, but got nothing.")
			}
			if ip, _ := GetIP([]string{"inet"}); ip == "" {
				t.Errorf("expected to find IP for inet, but found nothing.")
			}
			if ip, _ := GetIP([]string{"inet", lo}); ip == "127.0.0.1" {
				t.Errorf("expected to find inet ip, but found loopback instead")
			}
		}
	}
}

func TestInterfaceIpsLoopback(t *testing.T) {
	interfaces := make([]net.Interface, 1)

	interfaces[0] = net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  lo,
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
		if ip.Name != lo {
			t.Errorf("Expecting loopback interface, but got: %s", ip.Name)
		}
		if ip.IsIPv4() && ip.IPString() != "127.0.0.1" {
			t.Errorf("Expecting loopback interface [127.0.0.1] to be returned, got %s", ip.IPString())
		}
		if !ip.IsIPv4() && !strings.HasSuffix(ip.IPString(), "::1") {
			t.Errorf("Expecting loopback interface [::1] to be returned, got %s", ip.IPString())
		}
	}
}

func TestInterfaceIpsError(t *testing.T) {
	interfaces := make([]net.Interface, 2)

	interfaces[0] = net.Interface{
		Index: 1,
		MTU:   65536,
		Name:  lo,
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
		if ip.Name != lo {
			t.Errorf("Expecting loopback interface, but got: %s", ip.Name)
		}
		if ip.IsIPv4() && ip.IPString() != "127.0.0.1" {
			t.Errorf("Expecting loopback interface [127.0.0.1] to be returned, got %s", ip.IPString())
		}
		if !ip.IsIPv4() && !strings.HasSuffix(ip.IPString(), "::1") {
			t.Errorf("Expecting loopback interface [::1] to be returned, got %s", ip.IPString())
		}
	}
}

func TestInterfaceSpecParse(t *testing.T) {
	// Test Error Cases
	testSpecError(t, "")              // Nothing
	testSpecError(t, "!")             // Nonsense
	testSpecError(t, "127.0.0.1")     // No Network
	testSpecError(t, "eth0:inet5")    // Invalid IP Version
	testSpecError(t, "eth0[-1]")      // Invalid Index
	testSpecError(t, "static:abcdef") // Invalid IP

	// Test Interface Case
	testSpecInterfaceName(t, "eth0", "eth0", false, -1)
	testSpecInterfaceName(t, "eth0:inet6", "eth0", true, -1)
	testSpecInterfaceName(t, "eth0[1]", "eth0", false, 1)
	testSpecInterfaceName(t, "eth0[2]", "eth0", false, 2)
	testSpecInterfaceName(t, "inet", "*", false, -1)
	testSpecInterfaceName(t, "inet6", "*", true, -1)
	testSpecInterfaceName(t, "static:192.168.1.100", "static", false, 1)

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
	if name == "static" {
		staticSpec, ok := spec.(staticInterfaceSpec)
		if !ok {
			t.Errorf("Expected %s to parse as staticInterfaceSpec", spec)
			return
		}
		if staticSpec.Name != "static" {
			t.Errorf("Expected to parse interface name static but got %s", staticSpec.Name)
		}
		return
	}
	if index < 0 {
		inetSpec, ok := spec.(inetInterfaceSpec)
		if !ok {
			t.Errorf("Expected %s to parse as inetInterfaceSpec", spec)
			return
		}
		if inetSpec.Name != name {
			t.Errorf("Expected to parse interface name %s but got %s", name, inetSpec.Name)
		}
		if inetSpec.IPv6 != ipv6 {
			if ipv6 {
				t.Errorf("Expected spec %s to be IPv6", spec)
			} else {
				t.Errorf("Expected spec %s to be IPv4", spec)
			}
		}
		return
	}
	indexSpec, ok := spec.(indexInterfaceSpec)
	if !ok {
		t.Errorf("Expected %s to parse as indexInterfaceSpec", spec)
		return
	}
	if indexSpec.Name != name {
		t.Errorf("Expected to parse interface name %s but got %s", name, indexSpec.Name)
	}
	if indexSpec.Index != index {
		t.Errorf("Expected index to be %d but was %d", index, indexSpec.Index)
	}
}

func testSpecCIDR(t *testing.T, specStr string) {
	spec, err := parseInterfaceSpec(specStr)
	if err != nil {
		t.Errorf("Expected parse to succeed, but got error: %s", err)
	}
	cidrSpec, ok := spec.(cidrInterfaceSpec)
	if !ok {
		t.Errorf("Expected %s to parse as cidrInterfaceSpec", spec)
		return
	}
	if cidrSpec.Network == nil {
		t.Errorf("Expected spec to be a network CIDR")
	}
}

func TestFindIPWithSpecs(t *testing.T) {
	iips := getTestIPs()

	// Loopback
	testIPSpec(t, iips, "127.0.0.1", lo)
	testIPSpec(t, iips, "::1", lo6)

	// Static
	testIPSpec(t, iips, "192.168.1.100", "static:192.168.1.100")

	// Interface Name
	testIPSpec(t, iips, "10.2.0.1", "eth0")
	testIPSpec(t, iips, "10.2.0.1", "eth0:inet")
	testIPSpec(t, iips, "", "eth0:inet6")

	// Indexes
	testIPSpec(t, iips, "192.168.1.100", "eth0[1]")
	testIPSpec(t, iips, "", "eth0[2]")

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
		newInterfaceIP(lo6, "::1"),
		newInterfaceIP(lo, "127.0.0.1"),
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
		newInterfaceIP(lo, "::1"),
		newInterfaceIP(lo, "127.0.0.1"),
		newInterfaceIP("static:192.168.1.100", "192.168.1.100"),
	}
}

func newInterfaceIP(name string, ip string) interfaceIP {
	return interfaceIP{
		Name: name,
		IP:   net.ParseIP(ip),
	}
}
