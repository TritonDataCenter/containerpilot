package discovery

import (
	"fmt"
	"testing"

	consul "github.com/hashicorp/consul/api"
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

func TestWithConsul(t *testing.T) {
	testServer, err := NewTestServer(8500)
	if err != nil {
		t.Fatal(err)
	}
	defer testServer.Stop()

	testServer.WaitForAPI()

	t.Run("TestConsulTTLPass", testConsulTTLPass(testServer))
	t.Run("TestConsulRegisterWithInitialStatus", testConsulRegisterWithInitialStatus(testServer))
	t.Run("TestConsulReregister", testConsulReregister(testServer))
	t.Run("TestConsulCheckForChanges", testConsulCheckForChanges(testServer))
	t.Run("TestConsulEnableTagOverride", testConsulEnableTagOverride(testServer))
	t.Run("testConsulTagsMeta", testConsulTagsMeta(testServer))
}

func testConsulTTLPass(testServer *TestServer) func(*testing.T) {
	return func(t *testing.T) {
		consul, _ := NewConsul(testServer.HTTPAddr)
		name := "TestConsulTTLPass"
		service := generateServiceDefinition(name, consul)
		checkID := fmt.Sprintf("service:%s", service.ID)

		service.SendHeartbeat() // force registration and 1st heartbeat
		checks, _ := consul.Agent().Checks()
		check := checks[checkID]
		if check.Status != "passing" {
			t.Fatalf("status of check %s should be 'passing' but is %s", checkID, check.Status)
		}
	}
}

func testConsulRegisterWithInitialStatus(testServer *TestServer) func(*testing.T) {
	return func(t *testing.T) {
		consul, _ := NewConsul(testServer.HTTPAddr)
		name := "TestConsulRegisterWithInitialStatus"
		service := generateServiceDefinition(name, consul)
		checkID := fmt.Sprintf("service:%s", service.ID)

		service.RegisterWithInitialStatus() // force registration with initial status
		checks, _ := consul.Agent().Checks()
		check := checks[checkID]
		if check.Status != "warning" {
			t.Fatalf("status of check %s should be 'warning' but is %s", checkID, check.Status)
		}
	}
}

func testConsulReregister(testServer *TestServer) func(*testing.T) {
	return func(t *testing.T) {
		consul, _ := NewConsul(testServer.HTTPAddr)
		name := "TestConsulReregister"
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
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func testConsulTagsMeta(testServer *TestServer) func(*testing.T) {
	return func(t *testing.T) {
		consul, _ := NewConsul(testServer.HTTPAddr)
		name := "TestConsulReregister"
		service := generateServiceDefinition(name, consul)
		id := service.ID

		service.SendHeartbeat() // force registration and 1st heartbeat
		services, _ := consul.Agent().Services()
		svc := services[id]
		if !contains(svc.Tags, "a") || !contains(svc.Tags, "b") {
			t.Fatalf("first tag must containt a & b but is %s", svc.Tags)
		}
		if svc.Meta["keyA"] != "A" {
			t.Fatalf("first meta must containt keyA:A but is %s", svc.Meta["keyA"])
		}
		if svc.Meta["keyB"] != "B" {
			t.Fatalf("first meta must containt keyB:B but is %s", svc.Meta["keyB"])
		}

	}
}

func testConsulCheckForChanges(testServer *TestServer) func(*testing.T) {
	return func(t *testing.T) {
		backend := "TestConsulCheckForChanges"
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
		check := "service:TestConsulCheckForChanges"
		consul.Agent().UpdateTTL(check, "expired", "critical")
		if changed, _ := consul.CheckForUpstreamChanges(backend, "", ""); !changed {
			t.Errorf("%v should have changed after TTL expired.", id)
		}
	}
}

func testConsulEnableTagOverride(testServer *TestServer) func(*testing.T) {
	return func(t *testing.T) {
		backend := "TestConsulEnableTagOverride"
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
}

func generateServiceDefinition(serviceName string, consul *Consul) *ServiceDefinition {
	return &ServiceDefinition{
		ID:            serviceName,
		Name:          serviceName,
		IPAddress:     "192.168.1.1",
		InitialStatus: "warning",
		TTL:           5,
		Port:          9000,
		Consul:        consul,
		Tags:          []string{"a", "b"},
		Meta: map[string]string{
			"keyA": "A",
			"keyB": "B",
		},
	}
}
