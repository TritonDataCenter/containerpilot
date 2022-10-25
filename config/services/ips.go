package services

import (
	"bytes"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tritondatacenter/containerpilot/config/decode"
	log "github.com/sirupsen/logrus"
)

// IPFromInterfaces ...
func IPFromInterfaces(raw interface{}) (string, error) {
	interfaces, ifaceErr := decode.ToStrings(raw)
	if ifaceErr != nil {
		return "", ifaceErr
	}

	ipAddress, err := GetIP(interfaces)
	if err != nil {
		return "", err
	}
	return ipAddress, nil
}

// GetIP determines the IP address of the container
func GetIP(specList []string) (string, error) {

	if len(specList) == 0 {
		// Use a sane default
		specList = []string{"eth0:inet", "inet"}
	}

	specs, err := parseInterfaceSpecs(specList)
	if err != nil {
		return "", err
	}

	interfaces, interfacesErr := net.Interfaces()

	if interfacesErr != nil {
		return "", interfacesErr
	}

	interfaceIPs, interfaceIPsErr := getinterfaceIPs(interfaces)

	/* We had an error and there were no interfaces returned, this is clearly
	 * an error state. */
	if interfaceIPsErr != nil && len(interfaceIPs) < 1 {
		return "", interfaceIPsErr
	}
	/* We had error(s) and there were interfaces returned, this is potentially
	 * recoverable. Let's pass on the parsed interfaces and log the error
	 * state. */
	if interfaceIPsErr != nil && len(interfaceIPs) > 0 {
		log.Warnf("We had a problem reading information about some network "+
			"interfaces. If everything works, it is safe to ignore this "+
			"message. Details:\n%s\n", interfaceIPsErr)
	}

	return findIPWithSpecs(specs, interfaceIPs)
}

// findIPWithSpecs will use the given interface specification list and will
// find the first IP in the interfaceIPs that matches a spec
func findIPWithSpecs(specs []interfaceSpec, interfaceIPs []interfaceIP) (string, error) {
	// Find the interface matching the name given
	for _, spec := range specs {
		// Static IP given
		origSpec, ok := spec.(staticInterfaceSpec)
		if ok {
			return origSpec.IP.String(), nil
		}
		index := 0
		iface := ""
		for _, iip := range interfaceIPs {
			// Since the interfaces are ordered by name
			// a change in interface name can safely reset the index
			if iface != iip.Name {
				index = 0
				iface = iip.Name
			} else {
				index++
			}
			if spec.Match(index, iip) {
				return iip.IPString(), nil
			}
		}
	}

	// Interface not found, return error
	return "", fmt.Errorf("none of the interface specifications were able to match\nSpecifications: %s\nInterfaces IPs: %s",
		specs, interfaceIPs)
}

// Interface Spec
type interfaceSpec interface {
	Match(index int, iip interfaceIP) bool
}

// -- matches inet, inet6, interface:inet, and interface:inet6
type inetInterfaceSpec struct {
	Spec string
	Name string
	IPv6 bool
}

// -- matches static
type staticInterfaceSpec struct {
	Spec string
	Name string
	IP   net.IP
}

func (s staticInterfaceSpec) Match(index int, iip interfaceIP) bool {
	// Never matches
	return false
}

func (s inetInterfaceSpec) Match(index int, iip interfaceIP) bool {
	if s.Name != "*" && s.Name != iip.Name {
		return false
	}
	// Don't match loopback address for wildcard spec
	if s.Name == "*" && iip.IP.IsLoopback() {
		return false
	}
	return s.IPv6 != iip.IsIPv4()
}

// -- Indexed Interface Spec : eth0[1]
type indexInterfaceSpec struct {
	Spec  string
	Name  string
	Index int
}

func (spec indexInterfaceSpec) Match(index int, iip interfaceIP) bool {
	if spec.Name == iip.Name {
		return (spec.Index == index)
	}
	return false
}

// -- CIDR Interface Spec
type cidrInterfaceSpec struct {
	Spec    string
	Network *net.IPNet
}

func (spec cidrInterfaceSpec) Match(index int, iip interfaceIP) bool {
	return spec.Network.Contains(iip.IP)
}

func parseInterfaceSpecs(interfaces []string) ([]interfaceSpec, error) {
	var errors []string
	var specs []interfaceSpec
	for _, iface := range interfaces {
		spec, err := parseInterfaceSpec(iface)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}
		specs = append(specs, spec)
	}
	if len(errors) > 0 {
		err := fmt.Errorf(strings.Join(errors, "\n"))
		log.Errorln(err)
		return specs, err
	}
	return specs, nil
}

var (
	ifaceSpec = regexp.MustCompile(`^(?P<Name>\w+)(?:(?:\[(?P<Index>\d+)\])|(?::(?P<Version>inet6?)))?$`)
)

func parseInterfaceSpec(spec string) (interfaceSpec, error) {
	if spec == "inet" {
		return inetInterfaceSpec{Spec: spec, Name: "*", IPv6: false}, nil
	}
	if spec == "inet6" {
		return inetInterfaceSpec{Spec: spec, Name: "*", IPv6: true}, nil
	}
	if strings.HasPrefix(spec, "static:") {
		ip := strings.SplitAfter(spec, "static:")
		if _, err := strconv.Atoi(ip[1]); err != nil {
			nip := net.ParseIP(ip[1])
			if nip == nil {
				return nil, fmt.Errorf("unable to parse static ip %s in %s", ip[0], spec)
			}
			return staticInterfaceSpec{Spec: spec, Name: "static", IP: nip}, nil
		}
	}

	if match := ifaceSpec.FindStringSubmatch(spec); match != nil {
		name := match[1]
		index := match[2]
		inet := match[3]
		if index != "" {
			i, err := strconv.Atoi(index)
			if err != nil {
				return nil, fmt.Errorf("unable to parse index %s in %s", index, spec)
			}
			return indexInterfaceSpec{Spec: spec, Name: name, Index: i}, nil
		}
		if inet != "" {
			if inet == "inet" {
				return inetInterfaceSpec{Spec: spec, Name: name, IPv6: false}, nil
			}
			return inetInterfaceSpec{Spec: spec, Name: name, IPv6: true}, nil
		}
		return inetInterfaceSpec{Spec: spec, Name: name, IPv6: false}, nil
	}
	if _, net, err := net.ParseCIDR(spec); err == nil {
		return cidrInterfaceSpec{Spec: spec, Network: net}, nil
	}
	return nil, fmt.Errorf("unable to parse interface spec: %s", spec)
}

type interfaceIP struct {
	Name string
	IP   net.IP
}

func (iip interfaceIP) To16() net.IP {
	return iip.IP.To16()
}

func (iip interfaceIP) To4() net.IP {
	return iip.IP.To4()
}

func (iip interfaceIP) IsIPv4() bool {
	return iip.To4() != nil
}

func (iip interfaceIP) IPString() string {
	if v4 := iip.To4(); v4 != nil {
		return v4.String()
	}
	return iip.IP.String()
}

func (iip interfaceIP) String() string {
	return fmt.Sprintf("%s:%s", iip.Name, iip.IPString())
}

// Queries the network interfaces on the running machine and returns a list
// of IPs for each interface.
func getinterfaceIPs(interfaces []net.Interface) ([]interfaceIP, error) {
	var ifaceIPs []interfaceIP
	var errors []string

	for _, intf := range interfaces {
		ipAddrs, addrErr := intf.Addrs()

		if addrErr != nil {
			errors = append(errors, addrErr.Error())
			continue
		}

		for _, ipAddr := range ipAddrs {
			// Addresses some times come in the form "192.168.100.1/24 2001:DB8::/48"
			// so they must be split on whitespace
			for _, splitIP := range strings.Split(ipAddr.String(), " ") {
				ip, _, err := net.ParseCIDR(splitIP)
				if err != nil {
					errors = append(errors, err.Error())
					continue
				}
				intfIP := interfaceIP{Name: intf.Name, IP: ip}
				ifaceIPs = append(ifaceIPs, intfIP)
			}
		}
	}

	// Stable Sort the interface IPs so that selecting the correct IP in GetIP
	// can be consistent
	sort.Stable(ByInterfaceThenIP(ifaceIPs))

	/* If we had any errors parsing interfaces, we accumulate them all and
	 * then return them so that the caller can decide what they want to do. */
	if len(errors) > 0 {
		err := fmt.Errorf(strings.Join(errors, "\n"))
		log.Errorln(err)
		return ifaceIPs, err
	}

	return ifaceIPs, nil
}

// ByInterfaceThenIP implements the Sort with the following properties:
// 1. Sort interfaces alphabetically
// 2. Sort IPs by bytes (normalized to 16 byte form)
type ByInterfaceThenIP []interfaceIP

func (se ByInterfaceThenIP) Len() int      { return len(se) }
func (se ByInterfaceThenIP) Swap(i, j int) { se[i], se[j] = se[j], se[i] }
func (se ByInterfaceThenIP) Less(i, j int) bool {
	iip1, iip2 := se[i], se[j]
	if cmp := strings.Compare(iip1.Name, iip2.Name); cmp != 0 {
		return cmp < 0
	}
	return bytes.Compare(iip1.To16(), iip2.To16()) < 0
}
