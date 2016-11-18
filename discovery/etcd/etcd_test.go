package etcd

import (
	"reflect"
	"testing"
	"time"

	"github.com/coreos/etcd/client"
	"github.com/joyent/containerpilot/discovery"
	"golang.org/x/net/context"
)

func setupEtcd(serviceName string) (*Etcd, *discovery.ServiceDefinition) {
	etcd, _ := NewEtcdConfig(map[string]interface{}{"endpoints": []string{"http://etcd:4001"}})
	service := &discovery.ServiceDefinition{
		ID:        serviceName,
		Name:      serviceName,
		IPAddress: "192.168.1.1",
		TTL:       1,
		Port:      9000,
	}
	return etcd, service
}

func TestEtcdParseArrayEndpoints(t *testing.T) {
	cfg, _ := NewEtcdConfig(map[string]interface{}{"endpoints": []string{"http://etcd:4001"}})
	expected := []string{"http://etcd:4001"}
	actual := cfg.Client.Endpoints()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected endpoints %v but got %v", expected, actual)
	}
}

func TestEtcdParseStringEndpoints(t *testing.T) {
	cfg, _ := NewEtcdConfig(map[string]interface{}{"endpoints": "http://etcd:4001"})
	expected := []string{"http://etcd:4001"}
	actual := cfg.Client.Endpoints()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected endpoints %v but got %v", expected, actual)
	}
}

func TestEtcdTTLExpires(t *testing.T) {
	etcd, service := setupEtcd("service-TestEtcdTTLPass")
	id := service.ID

	etcd.SendHeartbeat(service) // force registration and TTL
	if !checkServiceHealthy(etcd, service) {
		t.Fatalf("Expected service %s to be registered and healthy, but was not", id)
	}

	time.Sleep(2 * time.Second) // wait for TTL to expire
	if checkServiceExists(etcd, service) {
		t.Fatalf("Expected service %s to be deregistered registered", id)
	}
}

func TestEtcdRegister(t *testing.T) {
	etcd, service := setupEtcd("service-TestEtcdRegister")
	id := service.ID

	// Should start off deregistered
	if checkServiceExists(etcd, service) {
		t.Fatalf("Expected service %s to be deregistered, but was not", id)
	}

	// Heartbeat should register
	etcd.SendHeartbeat(service)
	if !checkServiceExists(etcd, service) {
		t.Fatalf("Expected service %s to be registered, but was not", id)
	}

	// Explicit deregister should remove it
	etcd.Deregister(service)
	if checkServiceExists(etcd, service) {
		t.Fatalf("Expected service %s to be deregistered, but was not", id)
	}
}

func TestEtcdCheckForChanges(t *testing.T) {
	backend := "service-TestEtcdCheckForChanges"
	etcd, service := setupEtcd(backend)
	id := service.ID
	if etcd.CheckForUpstreamChanges(backend, "") {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	etcd.SendHeartbeat(service) // force registration and TTL

	if !etcd.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if etcd.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	time.Sleep(2 * time.Second) // wait for TTL to expire
	if !etcd.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
}

func checkServiceExists(etcd *Etcd, service *discovery.ServiceDefinition) bool {
	key := etcd.getNodeKey(service)
	if _, err := etcd.API.Get(context.Background(), key, nil); err != nil {
		if etcdErr, ok := err.(client.Error); ok {
			return etcdErr.Code != client.ErrorCodeKeyNotFound
		}
	}
	return true
}

func checkServiceHealthy(etcd *Etcd, service *discovery.ServiceDefinition) bool {
	key := etcd.getNodeKey(service)
	if resp, err := etcd.API.Get(context.Background(), key, nil); err != nil {
		if etcdErr, ok := err.(client.Error); ok {
			return etcdErr.Code != client.ErrorCodeKeyNotFound
		}
	} else {
		if len(resp.Node.Nodes) == 1 {
			return true
		}
	}
	return false
}
