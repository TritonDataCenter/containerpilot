package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DiscoveryService DiscoveryService
	PollTime         int
	HealthCheckExec  string
	OnChangeExec     string
}

// type alias to deal with parsing multiple -check params
type arrayFlags []string

func (i *arrayFlags) String() string { return strings.Join([]string(*i), ",") }
func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func parseArgs() *Config {
	var (
		discoveryUri    = flag.String("consul", "consul:8500", "Hostname and port for consul.")
		pollTime        = flag.Int("poll", 10, "Number of seconds to wait between polling health check")
		healthCheckExec = flag.String("health", "", "Executable to run to check the health of the application.")
		serviceName     = flag.String("name", "", "Name of service to register.")
		onChangeExec    = flag.String("onChange", "", "Executable to run when the discovery service has changes.")
		usePublicIP     = flag.Bool("public", false, "Publish the public IP rather than the private IP to the discovery service (default false)")
		toCheck         arrayFlags
		portArgs        arrayFlags
	)

	flag.Var(&portArgs, "port", "Port(s) to publish to the discovery service (accepts multiple).")
	flag.Var(&toCheck, "check", "Service(s) to check for changes (accepts multiple).")
	flag.Parse()

	ports, err := arrayAtoi(portArgs)
	if err != nil {
		log.Fatalf("Invalid -port argument(s): %s", portArgs)
	}

	// TODO: we need a better way to determine the right TTL
	ttl := *pollTime * 2
	hostname, _ := os.Hostname()

	config := &Config{
		DiscoveryService: NewConsulConfig(
			*discoveryUri,
			*serviceName,
			fmt.Sprintf("%s-%s", *serviceName, hostname),
			getIp(*usePublicIP),
			ports,
			ttl,
			toCheck),
		PollTime:        *pollTime,
		HealthCheckExec: *healthCheckExec,
		OnChangeExec:    *onChangeExec,
	}
	return config
}

func arrayAtoi(args arrayFlags) ([]int, error) {
	var ints = []int{}
	for _, i := range args {
		if j, err := strconv.Atoi(i); err != nil {
			return nil, err
		} else {
			ints = append(ints, j)
		}
	}
	return ints, nil
}

// determine the IP address of the container
func getIp(usePublic bool) string {
	interfaces, _ := net.Interfaces()
	var ips []net.IP
	_, loopback, _ := net.ParseCIDR("127.0.0.0/8")
	for _, intf := range interfaces {
		ipAddrs, _ := intf.Addrs()
		// We're assuming each interface has one IP here because neither Docker
		// nor Triton sets up IP aliasing.
		ipAddr, _, _ := net.ParseCIDR(ipAddrs[0].String())
		if !loopback.Contains(ipAddr) {
			ips = append(ips, ipAddr)
		}
	}
	var ip string
	for _, ipAddr := range ips {
		isPublic := isPublicIp(ipAddr)
		if isPublic && usePublic {
			ip = ipAddr.String()
			break
		} else if !isPublic && !usePublic {
			ip = ipAddr.String()
			break
		}
	}
	return ip
}

// parse an IPv4 address and return true if it's a public IP
func isPublicIp(ip net.IP) bool {
	_, c, _ := net.ParseCIDR("192.168.0.0/16")
	_, b, _ := net.ParseCIDR("172.16.0.0/12")
	_, a, _ := net.ParseCIDR("10.0.0.0/8")

	var privateNetworks = []*net.IPNet{c, b, a}
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return false
		}
	}
	return true
}
