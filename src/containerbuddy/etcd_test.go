package main

import (
	"testing"
	"time"
)

func setupEtcd(serviceName string) *Config {
	etcd := NewEtcdConfig(map[string]interface{}{
		"endpoints": []string{"http://etcd:4001"},
	})
	config := &Config{
		Services: []*ServiceConfig{
			&ServiceConfig{
				ID:               serviceName,
				Name:             serviceName,
				ipAddress:        "192.168.1.1",
				TTL:              1,
				Port:             9000,
				discoveryService: etcd,
			},
		},
		Backends: []*BackendConfig{
			&BackendConfig{
				Name:             serviceName,
				discoveryService: etcd,
			},
		},
	}
	return config
}

func TestEtcdTTLPass(t *testing.T) {
	config := setupEtcd("service-TestEtcdTTLPass")
	service := config.Services[0]
	etcd := service.discoveryService.(Etcd)
	id := service.ID

	service.SendHeartbeat() // force registration
	if !etcd.checkServiceExists(service) {
		t.Fatalf("Expected service %s to be registered, but was not", id)
	}

	service.SendHeartbeat() // write TTL and verify
	if !etcd.checkServiceExists(service) {
		t.Fatalf("Expected service %s to be registered, but was not", id)
	}

	time.Sleep(2 * time.Second)

	if etcd.checkServiceExists(service) {
		t.Fatalf("Expected service %s to be deregistered registered", id)
	}
}

func TestEtcdRegister(t *testing.T) {
	config := setupEtcd("service-TestEtcdRegister")
	service := config.Services[0]
	etcd := service.discoveryService.(Etcd)
	id := service.ID

	// Should start off deregistered
	if etcd.checkServiceExists(service) {
		t.Fatalf("Expected service %s to be deregistered, but was not", id)
	}

	// Heartbeat should register
	etcd.SendHeartbeat(service)
	if !etcd.checkServiceExists(service) {
		t.Fatalf("Expected service %s to be registered, but was not", id)
	}

	// Explicit deregister should remove it
	etcd.Deregister(service)
	if etcd.checkServiceExists(service) {
		t.Fatalf("Expected service %s to be deregistered, but was not", id)
	}
}

func TestEtcdCheckForChanges(t *testing.T) {
	config := setupEtcd("service-TestEtcdCheckForChanges")
	backend := config.Backends[0]
	service := config.Services[0]
	etcd := backend.discoveryService.(Etcd)
	id := service.ID
	if etcd.checkHealth(backend) {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	service.SendHeartbeat() // force registration
	service.SendHeartbeat() // write TTL

	if !etcd.checkHealth(backend) {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if etcd.checkHealth(backend) {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	time.Sleep(2 * time.Second) // wait for TTL to expire
	if !etcd.checkHealth(backend) {
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
