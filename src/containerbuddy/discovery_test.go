package main

import (
	"testing"
)

func setupConsul() *Config {
	return &Config{
		DiscoveryService: NewConsulConfig("consul:8500", "testService", 30,
			[]string{"upstream1", "upstream2"}),
		pollTime:        30,
		healthCheckExec: "/bin/true",
		onChangeExec:    "/bin/true",
	}
}

func TestTTLPass(t *testing.T) {
	config := setupConsul()
	config.DiscoveryService.WriteHealthCheck()
}
