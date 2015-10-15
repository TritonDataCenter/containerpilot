package main

import (
	"testing"
	"time"
)

func setupConsul(serviceName string) *Config {
	config := &Config{
		DiscoveryService: NewConsulConfig(
			"consul:8500",
			serviceName,
			serviceName,
			"192.168.1.1",
			[]int{8500, 9000},
			1, // ttl
			[]string{serviceName}),
		PollTime:        30,
		HealthCheckExec: "/bin/true",
		OnChangeExec:    "/bin/true",
	}
	return config
}

func TestTTLPass(t *testing.T) {
	config := setupConsul("service-TestTTLPass")
	consul := config.DiscoveryService.(Consul)
	id := consul.ServiceId

	config.DiscoveryService.WriteHealthCheck() // force registration
	checks, _ := consul.client.Agent().Checks()
	check := checks[id]
	if check.Status != "critical" {
		t.Fatalf("status of check %s should be 'critical' but is %s", id, check.Status)
	}

	config.DiscoveryService.WriteHealthCheck() // write TTL and verify
	checks, _ = consul.client.Agent().Checks()
	check = checks[id]
	if check.Status != "passing" {
		t.Fatalf("status of check %s should be 'passing' but is %s", id, check.Status)
	}
}

func TestCheckForChanges(t *testing.T) {
	config := setupConsul("service-TestCheckForChanges")
	consul := config.DiscoveryService.(Consul)
	id := consul.ServiceId
	if consul.checkHealth(id) {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	config.DiscoveryService.WriteHealthCheck() // force registration
	config.DiscoveryService.WriteHealthCheck() // write TTL

	if !consul.checkHealth(id) {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if consul.checkHealth(id) {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	time.Sleep(2 * time.Second) // wait for TTL to expire
	if !consul.checkHealth(id) {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
	config.DiscoveryService.WriteHealthCheck() // re-write TTL

	// switch to top-level caller to make sure we have test coverage of that loop
	if !consul.CheckForUpstreamChanges() {
		t.Errorf("%v should have changed after TTL re-entered.", id)
	}
	if consul.CheckForUpstreamChanges() {
		t.Errorf("%v should not have changed without TTL expiring.", id)
	}

}
