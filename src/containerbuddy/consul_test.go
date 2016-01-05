package main

import (
	"testing"
	"time"
)

func setupConsul(serviceName string) *Config {
	consul := NewConsulConfig("consul:8500")
	config := &Config{
		Services: []*ServiceConfig{
			&ServiceConfig{
				ID:               serviceName,
				Name:             serviceName,
				ipAddress:        "192.168.1.1",
				TTL:              1,
				Port:             9000,
				discoveryService: consul,
			},
		},
		Backends: []*BackendConfig{
			&BackendConfig{
				Name:             serviceName,
				discoveryService: consul,
			},
		},
	}
	return config
}

func TestConsulTTLPass(t *testing.T) {
	config := setupConsul("service-TestConsulTTLPass")
	service := config.Services[0]
	consul := service.discoveryService.(Consul)
	id := service.ID

	service.SendHeartbeat() // force registration
	checks, _ := consul.Agent().Checks()
	check := checks[id]
	if check.Status != "critical" {
		t.Fatalf("status of check %s should be 'critical' but is %s", id, check.Status)
	}

	service.SendHeartbeat() // write TTL and verify
	checks, _ = consul.Agent().Checks()
	check = checks[id]
	if check.Status != "passing" {
		t.Fatalf("status of check %s should be 'passing' but is %s", id, check.Status)
	}
}

func TestConsulCheckForChanges(t *testing.T) {
	config := setupConsul("service-TestConsulCheckForChanges")
	backend := config.Backends[0]
	service := config.Services[0]
	consul := backend.discoveryService.(Consul)
	id := service.ID
	if consul.checkHealth(*backend) {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	service.SendHeartbeat() // force registration
	service.SendHeartbeat() // write TTL

	if !consul.checkHealth(*backend) {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if consul.checkHealth(*backend) {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	time.Sleep(2 * time.Second) // wait for TTL to expire
	if !consul.checkHealth(*backend) {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
	service.SendHeartbeat() // re-write TTL

	// switch to top-level caller to make sure we have test coverage there
	if !backend.CheckForUpstreamChanges() {
		t.Errorf("%v should have changed after TTL re-entered.", id)
	}
	if backend.CheckForUpstreamChanges() {
		t.Errorf("%v should not have changed without TTL expiring.", id)
	}

}
