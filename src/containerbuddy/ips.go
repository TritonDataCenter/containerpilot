package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

// GetIP determines the IP address of the container
func GetIP(interfaceNames []string) (string, error) {

	if interfaceNames == nil || len(interfaceNames) == 0 {
		// Use a sane default
		interfaceNames = []string{"eth0"}
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
		log.Printf("We had a problem reading information about some network "+
			"interfaces. If everything works, it is safe to ignore this"+
			"message. Details:\n%s\n", interfaceIPsErr)
	}

	// Find the interface matching the name given
	for _, interfaceName := range interfaceNames {
		for _, intf := range interfaceIPs {
			if interfaceName == intf.Name {
				return intf.IP, nil
			}
		}
	}

	// Interface not found, return error
	return "", fmt.Errorf("Unable to find interfaces %s in %#v",
		interfaceNames, interfaceIPs)
}

type interfaceIP struct {
	Name string
	IP   string
}

// Queries the network interfaces on the running machine and returns a list
// of IPs for each interface. Currently, this only returns IPv4 addresses.
func getinterfaceIPs(interfaces []net.Interface) ([]interfaceIP, error) {
	var ifaceIPs []interfaceIP
	var errors []string

	for _, intf := range interfaces {
		ipAddrs, addrErr := intf.Addrs()

		if addrErr != nil {
			errors = append(errors, addrErr.Error())
			continue
		}

		/* As crazy as it may seem, yes you can have an interface that doesn't
		 * have an IP address assigned. */
		if len(ipAddrs) == 0 {
			continue
		}

		/* We ignore aliases for the time being. We assume that that
		 * authoritative address is the first address returned from the
		 * interface. */
		ifaceIP, parsingErr := parseIPFromAddress(ipAddrs[0], intf)

		if parsingErr != nil {
			errors = append(errors, parsingErr.Error())
			continue
		}

		ifaceIPs = append(ifaceIPs, ifaceIP)
	}

	/* If we had any errors parsing interfaces, we accumulate them all and
	 * then return them so that the caller can decide what they want to do. */
	if len(errors) > 0 {
		err := fmt.Errorf(strings.Join(errors, "\n"))
		println(err.Error())
		return ifaceIPs, err
	}

	return ifaceIPs, nil
}

// Parses an IP and interface name out of the provided address and interface
// objects. We assume that the default IPv4 address will be the first IPv4 address
// to appear in the list of IPs presented for the interface.
func parseIPFromAddress(address net.Addr, intf net.Interface) (interfaceIP, error) {
	ips := strings.Split(address.String(), " ")

	// In Linux, we will typically see a value like:
	// 192.168.0.7/24 fe80::12c3:7bff:fe45:a2ff/64

	var ipv4 string
	ipv4Regex := "^\\d+\\.\\d+\\.\\d+\\.\\d+.*$"

	for _, ip := range ips {
		matched, matchErr := regexp.MatchString(ipv4Regex, ip)

		if matchErr != nil {
			return interfaceIP{}, matchErr
		}

		if matched {
			ipv4 = ip
			break
		}
	}

	if len(ipv4) < 1 {
		msg := fmt.Sprintf("No parsable IPv4 address was available for "+
			"interface: %s", intf.Name)
		return interfaceIP{}, errors.New(msg)
	}

	ipAddr, _, parseErr := net.ParseCIDR(ipv4)

	if parseErr != nil {
		return interfaceIP{}, parseErr
	}

	ifaceIP := interfaceIP{Name: intf.Name, IP: ipAddr.String()}

	return ifaceIP, nil
}
