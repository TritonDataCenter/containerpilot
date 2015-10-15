package main

import (
	"testing"
)

func setupConsul() *Config {
	config := &Config{
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
	return config
}

func TestTTLPass(t *testing.T) {
	config := setupConsul()
	consul := config.DiscoveryService.(Consul)
	id := consul.ServiceId

	config.DiscoveryService.WriteHealthCheck() // force registration
	checks, _ := consul.client.Agent().Checks()
	check := checks[id]
	if check.Status != "critical" {
		t.Errorf("status of check %s should be 'critical' but is %s", id, check.Status)
	}

	config.DiscoveryService.WriteHealthCheck() // write TTL and verify
	checks, _ = consul.client.Agent().Checks()
	check = checks[id]
	if check.Status != "passing" {
		t.Errorf("status of check %s should be 'passing' but is %s", id, check.Status)
	}
}
