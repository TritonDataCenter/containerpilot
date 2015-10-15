package main

import (
	"flag"
	"net"
	"os"
	"testing"
)

func TestArgParse(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = nil
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"this", "-poll", "20", "/root/examples/test.sh", "doStuff", "--debug"}
	config := parseArgs()
	if config.PollTime != 20 {
		t.Errorf("Expected PollTime to be 20 but got: %d", config.PollTime)
	}
	args := flag.Args()
	if len(args) != 3 || args[0] != "/root/examples/test.sh" {
		t.Errorf("Expected 3 args but got unexpected results: %v", args)
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
