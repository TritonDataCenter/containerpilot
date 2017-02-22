package consul

import (
	"fmt"
	"testing"

	"github.com/hashicorp/consul/testutil"
	"github.com/joyent/containerpilot/discovery"
)

func TestConsulObjectParse(t *testing.T) {
	rawCfg := map[string]interface{}{
		"address": "consul:8501",
		"scheme":  "https",
		"token":   "ec492475-7753-4ff0-bd65-2f056d68f78b",
	}
	_, err := NewConsulConfig(rawCfg)
	if err != nil {
		t.Fatalf("Unable to parse config: %v", err)
	}
}

func TestConsulAddressParse(t *testing.T) {
	// typical valid entries
	runParseTest(t, "https://consul:8500", "consul:8500", "https")
	runParseTest(t, "http://consul:8500", "consul:8500", "http")
	runParseTest(t, "consul:8500", "consul:8500", "http")

	// malformed URI: we won't even try to fix these and just let them bubble up
	// to the Consul API call where it'll fail there.
	runParseTest(t, "httpshttps://consul:8500", "httpshttps://consul:8500", "http")
	runParseTest(t, "https://https://consul:8500", "https://consul:8500", "https")
	runParseTest(t, "http://https://consul:8500", "https://consul:8500", "http")
	runParseTest(t, "consul:8500https://", "consul:8500https://", "http")
	runParseTest(t, "", "", "http")
}

func runParseTest(t *testing.T, uri, expectedAddress, expectedScheme string) {

	address, scheme := parseRawURI(uri)
	if address != expectedAddress || scheme != expectedScheme {
		t.Fatalf("Expected %s over %s but got %s over %s",
			expectedAddress, expectedScheme, address, scheme)
	}
}

/*
The TestWithConsul suite of tests uses Hashicorp's own testutil for managing
a Consul server for testing. The 'consul' binary must be in the $PATH
ref https://github.com/hashicorp/consul/tree/master/testutil
*/

var testServer *testutil.TestServer

func TestWithConsul(t *testing.T) {
	testServer = testutil.NewTestServerConfig(t, func(c *testutil.TestServerConfig) {
		c.LogLevel = "err"
	})
	defer testServer.Stop()
	t.Run("TestConsulTTLPass", testConsulTTLPass)
	t.Run("TestConsulReregister", testConsulReregister)
	t.Run("TestConsulCheckForChanges", testConsulCheckForChanges)
	t.Run("TestConsulEnableTagOverride", testConsulEnableTagOverride)
}

func testConsulTTLPass(t *testing.T) {
	consul, _ := NewConsulConfig(testServer.HTTPAddr)
	service := generateServiceDefinition(fmt.Sprintf("service-TestConsulTTLPass"))
	id := service.ID

	consul.SendHeartbeat(service) // force registration and 1st heartbeat
	checks, _ := consul.Agent().Checks()
	check := checks[id]
	if check.Status != "passing" {
		t.Fatalf("status of check %s should be 'passing' but is %s", id, check.Status)
	}
}

func testConsulReregister(t *testing.T) {
	consul, _ := NewConsulConfig(testServer.HTTPAddr)
	service := generateServiceDefinition(fmt.Sprintf("service-TestConsulReregister"))
	id := service.ID
	consul.SendHeartbeat(service) // force registration and 1st heartbeat
	services, _ := consul.Agent().Services()
	svc := services[id]
	if svc.Address != "192.168.1.1" {
		t.Fatalf("service address should be '192.168.1.1' but is %s", svc.Address)
	}

	// new Consul client (as though we've restarted)
	consul, _ = NewConsulConfig(testServer.HTTPAddr)
	service.IPAddress = "192.168.1.2"
	consul.SendHeartbeat(service) // force re-registration and 1st heartbeat

	services, _ = consul.Agent().Services()
	svc = services[id]
	if svc.Address != "192.168.1.2" {
		t.Fatalf("service address should be '192.168.1.2' but is %s", svc.Address)
	}
}

func testConsulCheckForChanges(t *testing.T) {
	backend := fmt.Sprintf("service-TestConsulCheckForChanges")
	consul, _ := NewConsulConfig(testServer.HTTPAddr)
	service := generateServiceDefinition(backend)
	id := service.ID
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	consul.SendHeartbeat(service) // force registration and 1st heartbeat

	if !consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	consul.Agent().UpdateTTL(id, "expired", "critical")
	if !consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
}

func testConsulEnableTagOverride(t *testing.T) {
	backend := fmt.Sprintf("service-TestConsulEnableTagOverride")
	consul, _ := NewConsulConfig(testServer.HTTPAddr)
	service := &discovery.ServiceDefinition{
		ID:        backend,
		Name:      backend,
		IPAddress: "192.168.1.1",
		TTL:       1,
		Port:      9000,
		ConsulExtras: &discovery.ConsulExtras{
			EnableTagOverride: true,
		},
	}
	id := service.ID
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	consul.SendHeartbeat(service) // force registration
	catalogService, _, err := consul.Catalog().Service(id, "", nil)
	if err != nil {
		t.Fatalf("Error finding service: %v", err)
	}

	for _, service := range catalogService {
		if service.ServiceEnableTagOverride != true {
			t.Errorf("%v should have had EnableTagOverride set to true", id)
		}
	}
}

func generateServiceDefinition(serviceName string) *discovery.ServiceDefinition {
	return &discovery.ServiceDefinition{
		ID:        serviceName,
		Name:      serviceName,
		IPAddress: "192.168.1.1",
		TTL:       5,
		Port:      9000,
	}
}
