package consul

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joyent/containerpilot/discovery"
)

func getTestConsulAddress() string {
	addr := os.Getenv("CONSUL")
	if addr == "" {
		addr = "localhost:8500"
	}
	fmt.Println(addr)
	return addr
}

func setupConsul(serviceName string) (*Consul, *discovery.ServiceDefinition) {
	consul, _ := NewConsulConfig(getTestConsulAddress())
	service := &discovery.ServiceDefinition{
		ID:        serviceName,
		Name:      serviceName,
		IPAddress: "192.168.1.1",
		TTL:       1,
		Port:      9000,
	}
	return consul, service
}

func setupWaitForLeader(consul *Consul) error {
	maxRetry := 30
	retry := 0
	var err error

	// we need to wait for Consul to start and self-elect
	for ; retry < maxRetry; retry++ {
		if retry > 0 {
			time.Sleep(1 * time.Second)
		}
		if leader, err := consul.Status().Leader(); err == nil && leader != "" {
			break
		}
	}
	return err
}

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

func TestConsulTTLPass(t *testing.T) {
	consul, service := setupConsul("service-TestConsulTTLPass")
	if err := setupWaitForLeader(consul); err != nil {
		t.Errorf("Consul leader could not be elected.")
	}
	id := service.ID

	consul.SendHeartbeat(service) // force registration and 1st heartbeat
	checks, _ := consul.Agent().Checks()
	check := checks[id]
	if check.Status != "passing" {
		t.Fatalf("status of check %s should be 'passing' but is %s", id, check.Status)
	}
}

func TestConsulReregister(t *testing.T) {
	consul, service := setupConsul("service-TestConsulReregister")
	if err := setupWaitForLeader(consul); err != nil {
		t.Errorf("Consul leader could not be elected.")
	}
	id := service.ID
	consul.SendHeartbeat(service) // force registration and 1st heartbeat
	services, _ := consul.Agent().Services()
	svc := services[id]
	if svc.Address != "192.168.1.1" {
		t.Fatalf("service address should be '192.168.1.1' but is %s", svc.Address)
	}

	// new Consul client (as though we've restarted)
	consul, service = setupConsul("service-TestConsulReregister")
	service.IPAddress = "192.168.1.2"
	consul.SendHeartbeat(service) // force re-registration and 1st heartbeat

	services, _ = consul.Agent().Services()
	svc = services[id]
	if svc.Address != "192.168.1.2" {
		t.Fatalf("service address should be '192.168.1.2' but is %s", svc.Address)
	}
}

func TestConsulCheckForChanges(t *testing.T) {
	backend := "service-TestConsulCheckForChanges"
	consul, service := setupConsul(backend)
	if err := setupWaitForLeader(consul); err != nil {
		t.Errorf("Consul leader could not be elected.")
	}
	id := service.ID
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Fatalf("First read of %s should show `false` for change", id)
	}
	consul.SendHeartbeat(service) // force registration
	consul.SendHeartbeat(service) // write TTL

	if !consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after first health check TTL", id)
	}
	if consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should not have changed without TTL expiring", id)
	}
	time.Sleep(2 * time.Second) // wait for TTL to expire
	if !consul.CheckForUpstreamChanges(backend, "") {
		t.Errorf("%v should have changed after TTL expired.", id)
	}
}

func TestConsulEnableTagOverride(t *testing.T) {
	backend := "service-TestConsulEnableTagOverride"
	consul, _ := NewConsulConfig(getTestConsulAddress())
	if err := setupWaitForLeader(consul); err != nil {
		t.Errorf("Consul leader could not be elected.")
	}
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
