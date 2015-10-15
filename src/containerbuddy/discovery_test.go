package main

import (
	"testing"
)

func setupConsul() *Config {
	return &Config{
		DiscoveryService: NewConsulConfig(
			"consul:8500",
			"testService",
			"192.168.1.1",
			[]int{8500, 9000},
			60, // ttl
			[]string{"upstream1", "upstream2"}),
		PollTime:        30,
		HealthCheckExec: "/bin/true",
		OnChangeExec:    "/bin/true",
	}
}

func TestTTLPass(t *testing.T) {
	config := setupConsul()
	config.DiscoveryService.WriteHealthCheck()
}
