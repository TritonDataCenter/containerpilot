package discovery

import (
	"fmt"
	"testing"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/testutil"
	"github.com/stretchr/testify/assert"
)

func TestConsulObjectParse(t *testing.T) {
	rawCfg := map[string]interface{}{
		"address": "consul:8501",
		"scheme":  "https",
		"token":   "ec492475-7753-4ff0-bd65-2f056d68f78b",
	}
	_, err := NewConsul(rawCfg)
	if err != nil {
		t.Fatalf("unable to parse config: %v", err)
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

func TestCheckForChanges(t *testing.T) {
	c, _ := NewConsul(`consul: "localhost:8500"`)

	t0 := []*consul.ServiceEntry{}
	didChange := c.compareAndSwap("test", t0)
	assert.False(t, didChange, "value for 'didChange' after t0")

	t1 := []*consul.ServiceEntry{
		{Service: &consul.AgentService{Address: "1.2.3.4", Port: 80}},
		{Service: &consul.AgentService{Address: "1.2.3.5", Port: 80}},
	}
	didChange = c.compareAndSwap("test", t1)
	assert.True(t, didChange, "value for 'didChange' after t1")

	didChange = c.compareAndSwap("test", t0)
	assert.True(t, didChange, "value for 'didChange' after t0 (again)")

	didChange = c.compareAndSwap("test", t1)
	assert.True(t, didChange, "value for 'didChange' after t1 (again)")

	t3 := []*consul.ServiceEntry{
		{Service: &consul.AgentService{Address: "1.2.3.4", Port: 80}}}
	didChange = c.compareAndSwap("test", t3)
	assert.True(t, didChange, "value for 'didChange' after t3")
}

/*
The TestWithConsul suite of tests uses Hashicorp's own testutil for managing
a Consul server for testing. The 'consul' binary must be in the $PATH
ref https://github.com/hashicorp/consul/tree/master/testutil
*/

var testServer *testutil.TestServer

func TestWithConsul(t *testing.T) {
	testServer, _ = testutil.NewTestServerConfigT(t, func(c *testutil.TestServerConfig) {
		c.LogLevel = "err"
	})
	defer testServer.Stop()
	t.Run("TestConsulTTLPass", testConsulTTLPass)
	t.Run("TestConsulReregister", testConsulReregister)
	t.Run("TestConsulCheckForChanges", testConsulCheckForChanges)
	t.Run("TestConsulEnableTagOverride", testConsulEnableTagOverride)
}

func testConsulTTLPass(t *testing.T) {
	consul, _ := NewConsul(testServer.HTTPAddr)
	name := fmt.Sprintf("TestConsulTTLPass")
	service := generateServiceDefinition(name, consul)
	checkID := fmt.Sprintf("service:%s", service.ID)

	service.SendHeartbeat() // force registration and 1st heartbeat
	checks, _ := consul.Agent().Checks()
	check := checks[checkID]
	if check.Status != "passing" {
		t.Fatalf("status of check %s should be 'passing' but is %s", checkID, check.Status)
	}
}

func testConsulReregister(t *testing.T) {
	consul, _ := NewConsul(testServer.HTTPAddr)
	name := fmt.Sprintf("TestConsulReregister")
	service := generateServiceDefinition(name, consul)
	id := service.ID

	service.SendHeartbeat() // force registration and 1st heartbeat
	services, _ := consul.Agent().Services()
	svc := services[id]
	if svc.Address != "192.168.1.1" {
		t.Fatalf("service address should be '192.168.1.1' but is %s", svc.Address)
	}

	// new Consul client (as though we've restarted)
	consul, _ = NewConsul(testServer.HTTPAddr)
	service = generateServiceDefinition(name, consul)
	service.IPAddress = "192.168.1.2"
	service.SendHeartbeat() // force re-registration and 1st heartbeat

	services, _ = consul.Agent().Services()
	svc = services[id]
	if svc.Address != "192.168.1.2" {
		t.Fatalf("service address should be '192.168.1.2' but is %s", svc.Address)
	}
}

func testConsulCheckForChanges(t *testing.T) {
	backend := fmt.Sprintf("TestConsulCheckForChanges")
	consul, _ := NewConsul(testServer.HTTPAddr)
	service := generateServiceDefinition(backend, consul)
	id := service.ID
	if changed, _ := consul.CheckForUpstreamChanges(backend, "", ""); changed {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	service.SendHeartbeat() // force registration and 1st heartbeat

	if changed, _ := consul.CheckForUpstreamChanges(backend, "", ""); !changed {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if changed, _ := consul.CheckForUpstreamChanges(backend, "", ""); changed {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	check := fmt.Sprintf("service:TestConsulCheckForChanges")
	consul.Agent().UpdateTTL(check, "expired", "critical")
	if changed, _ := consul.CheckForUpstreamChanges(backend, "", ""); !changed {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
}

func testConsulEnableTagOverride(t *testing.T) {
	backend := fmt.Sprintf("TestConsulEnableTagOverride")
	consul, _ := NewConsul(testServer.HTTPAddr)
	service := &ServiceDefinition{
		ID:                backend,
		Name:              backend,
		IPAddress:         "192.168.1.1",
		TTL:               1,
		Port:              9000,
		EnableTagOverride: true,
		Consul:            consul,
	}
	id := service.ID
	if changed, _ := consul.CheckForUpstreamChanges(backend, "", ""); changed {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	service.SendHeartbeat() // force registration
	catalogService, _, err := consul.Catalog().Service(id, "", nil)
	if err != nil {
		t.Fatalf("error finding service: %v", err)
	}

	for _, service := range catalogService {
		if service.ServiceEnableTagOverride != true {
			t.Errorf("%v should have had EnableTagOverride set to true", id)
		}
	}
}

func generateServiceDefinition(serviceName string, consul *Consul) *ServiceDefinition {
	return &ServiceDefinition{
		ID:        serviceName,
		Name:      serviceName,
		IPAddress: "192.168.1.1",
		TTL:       5,
		Port:      9000,
		Consul:    consul,
	}
}
