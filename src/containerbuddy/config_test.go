package main

import (
	"flag"
	"net"
	"os"
	"testing"
)

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

func TestConfigParse(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = nil
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"this", "-config", testJson, "/root/examples/test.sh", "doStuff", "--debug"}
	config := loadConfig()

	if len(config.Backends) != 2 || len(config.Services) != 2 {
		t.Errorf("Expected 2 backends and 2 services but got: %v", config)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/root/examples/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
	}
	if config.OnStart != "/bin/to/onStart.sh" {
		t.Errorf("onStart not configured")
	}
}

func TestGetIp(t *testing.T) {
	if ip := getIp(false); ip == "" {
		t.Errorf("Expected private IP but got nothing")
	}
	if ip := getIp(true); ip != "" {
		t.Errorf("Expected no public IP but got: %s", ip)
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
