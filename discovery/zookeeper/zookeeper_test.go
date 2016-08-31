package zookeeper

import (
	"bytes"
	"github.com/joyent/containerpilot/discovery"
	"github.com/samuel/go-zookeeper/zk"
	"testing"
	"time"
)

// Factories, utilities
func zkConnection() *zk.Conn {
	c, _, _ := zk.Connect([]string{"127.0.0.1"}, time.Second)
	return c
}

func serviceDef(id string) *discovery.ServiceDefinition {
	return &discovery.ServiceDefinition{
		ID:        id,
		Name:      "my-service",
		IPAddress: "192.168.1.1",
		TTL:       1,
		Port:      9000,
	}
}

func zookeeper() *ZooKeeper {
	return &ZooKeeper{
		Client: zkConnection(),
		Prefix: "/containerpilot",
	}
}

// Characterization tests.
func TestZKCreateNode(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	_, err := c.Create(
		"/node",                     // path
		[]byte("value, can be nil"), // data
		zk.FlagEphemeral,            // flags
		zk.WorldACL(zk.PermAll))     // ACL
	if err != nil {
		t.Fatalf("Unable to create node")
	}
}

func TestCreateIntermediateEphemeralNode(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	_, err := c.Create("/a/b/c", nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err == nil {
		t.Fatalf("Should not be able to create a full, ephemeral path")
	}
}

func TestEphemeralParent(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	_, err := c.Create("/a", nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	_, err = c.Create("/a/b", nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err == nil {
		t.Fatalf("Should not be able to add a node to an ephemeral parent: %s", err)
	}
}

func TestCreateIntermediatePermanentNode(t *testing.T) {
	c := zkConnection()
	path := "/a/b/c"
	defer c.Close()
	_, err := c.Create(path, nil, 0, zk.WorldACL(zk.PermAll))
	if err == nil {
		t.Fatalf("Should not be able to create a full, permanent path")
	}
}

func TestCreateFullPath(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	_, err := c.Create("/a", nil, 0, zk.WorldACL(zk.PermAll))
	_, err = c.Create("/a/b", nil, 0, zk.WorldACL(zk.PermAll))
	_, err = c.Create("/a/b/c", nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		t.Fatalf("Unable to create a full path, step by step: %s", err)
	}
	c.Delete("/a/b/c", -1)
	c.Delete("/a/b", -1)
	c.Delete("/a", -1)
}

func TestCreateParentPath(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	zookeeper := &ZooKeeper{Client: c, Prefix: "<doesn’t matter>"}
	path := "/a/b/c"
	err := zookeeper.createParentPath(path)
	if err != nil {
		t.Fatalf("Unable to create parent path: %s", err)
	}
	if exists, _, _ := c.Exists(path); !exists {
		t.Fatalf("Path %s not created, %s", path, err)
	}
	c.Delete("/a/b/c", -1)
	c.Delete("/a/b", -1)
	c.Delete("/a", -1)
}

func TestCreateParentPathShouldIgnoreExistingIntermediateNodes(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	c.Create("/a", nil, 0, zk.WorldACL(zk.PermAll))
	zookeeper := &ZooKeeper{Client: c, Prefix: "<doesn’t matter>"}
	path := "/a/b/c"
	err := zookeeper.createParentPath("/a/b/c")
	if err != nil {
		t.Fatalf("Unable to create parent path: %s", err)
	}
	if exists, _, _ := c.Exists(path); !exists {
		t.Fatalf("Path %s not created, %s", path, err)
	}
	c.Delete("/a/b/c", -1)
	c.Delete("/a/b", -1)
	c.Delete("/a", -1)
}

func TestZKCreateNodeIdempotency(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	_, err := c.Create("/node", []byte("v"), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		t.Fatalf("Unable to create node")
	}
	_, err = c.Create("/node", []byte("v2"), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err == nil {
		t.Fatalf("Create node should not be idempotent")
	}
}

func TestZKConnectionEventsLazyness(t *testing.T) {
	c, ch, _ := zk.Connect([]string{"127.0.0.1"}, time.Second)
	var events []zk.Event
	go func() {
		for event := range ch {
			events = append(events, event)
		}
	}()
	defer c.Close()
	if len(events) > 0 {
		t.Fatalf("No events should be emitted without activity. %+v", events)
	}
}

func TestZKNodeCreationShouldNotEmitEventsWithSessionChan(t *testing.T) {
	c, ch, _ := zk.Connect([]string{"127.0.0.1"}, time.Second)
	var events []zk.Event
	go func() {
		for event := range ch {
			events = append(events, event)
		}
	}()
	defer c.Close()
	c.Create("/node", nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if len(events) != 3 {
		t.Fatalf("Unexpected number of events, %d", len(events))
	}
	if events[0].Type != zk.EventSession ||
		events[1].Type != zk.EventSession ||
		events[2].Type != zk.EventSession {
		t.Error("All the events should of type EventSession")
	}
}

func TestZKWatchEventNodeDataChanged(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	c.Create("/node", nil, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	_, _, ch, _ := c.GetW("/node")
	var events []zk.Event
	go func() {
		for event := range ch {
			events = append(events, event)
		}
	}()
	c.Set("/node", []byte("value"), -1)
	c.Set("/node", []byte("another value"), -1)
	if len(events) != 1 {
		t.Fatalf("Watchers receive only the first event after they are set, %d", len(events))
	}
	if events[0].Type != zk.EventNodeDataChanged {
		t.Error("All the events should of type EventNodeDataChanged")
	}
}

func TestZKReadNode(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	c.Create("/node", []byte("v"), zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	data, _, err := c.Get("/node")
	if err != nil {
		t.Fatalf("Unable to read node")
	}
	if string(data) != "v" {
		t.Fatalf("Unexpected data %s", data)
	}
}

// ContainerPilot tests
func TestNewZooKeeperConfig(t *testing.T) {
	rawCfg := map[string]interface{}{
		"address": "127.0.0.1",
	}
	if _, err := NewZooKeeperConfig(rawCfg); err != nil {
		t.Fatalf("Unable to parse config: %s", err)
	}
}

func TestEncodeZKNodeValue(t *testing.T) {
	s := "my-service"
	ip := "192.168.1.1"
	ttl := 1
	p := 9000
	service := &discovery.ServiceDefinition{
		ID:        s,
		Name:      s,
		IPAddress: ip,
		TTL:       ttl,
		Port:      p,
	}
	expectedResult := []byte(`{"id":"my-service","name":"my-service","address":"192.168.1.1","port":9000,"tags":null}`)
	encodedServiceDef := encodeZooKeeperNodeValue(service)
	if !bytes.Equal(encodedServiceDef, expectedResult) {
		t.Fatalf("Unexpected service encoding %s", encodedServiceDef)
	}
}

func TestDecodeZKNodeValue(t *testing.T) {
	s := "my-service"
	ip := "192.168.1.1"
	p := 9000
	serviceNode := ServiceNode{
		ID:      s,
		Name:    s,
		Address: ip,
		Port:    p,
	}
	encodedService := `{"id":"my-service","name":"my-service","address":"192.168.1.1","port":9000,"tags":null}`
	decodedService, _ := decodeZooKeeperNodeValue([]byte(encodedService))
	if serviceNode.ID != decodedService.ID ||
		serviceNode.Name != decodedService.Name ||
		serviceNode.Address != decodedService.Address ||
		serviceNode.Port != decodedService.Port {
		t.Fatalf("Unexpected service decoding %s", decodedService)
	}
}

func TestNodeKey(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	zookeeper := &ZooKeeper{
		Client: c,
		Prefix: "/containerpilot",
	}
	s := "my-service"
	id := "srv-id"
	ip := "192.168.1.1"
	ttl := 1
	p := 9000
	service := &discovery.ServiceDefinition{
		ID:        id,
		Name:      s,
		IPAddress: ip,
		TTL:       ttl,
		Port:      p,
	}
	if zookeeper.getNodeKey(service) != "/containerpilot/my-service/srv-id" {
		t.Fatalf("Unexpected node key %s", zookeeper.getNodeKey(service))
	}
}

func TestAppKey(t *testing.T) {
	c := zkConnection()
	defer c.Close()
	zookeeper := &ZooKeeper{
		Client: c,
		Prefix: "/containerpilot",
	}
	if a := zookeeper.getAppKey("my-app"); a != "/containerpilot/my-app" {
		t.Fatalf("Unexpected app key %s", a)
	}
}

func TestRegisterService(t *testing.T) {
	expectedValue := `{"id":"srv-id","name":"my-service","address":"192.168.1.1","port":9000,"tags":null}`
	zookeeper := zookeeper()
	defer zookeeper.Client.Close()
	defer zookeeper.Client.Delete("/containerpilot/my-service/srv-id", -1)
	defer zookeeper.Client.Delete("/containerpilot/my-service", -1)
	defer zookeeper.Client.Delete("/containerpilot", -1)

	service := serviceDef("srv-id")
	err := zookeeper.registerService(service)
	if err != nil {
		t.Fatalf("Unable to register service: %s", err)
	}
	data, _, err := zookeeper.Client.Get("/containerpilot/my-service/srv-id")
	if err != nil {
		t.Fatalf("Unable to read node")
	}
	if string(data) != expectedValue {
		t.Fatalf("Unexpected data %s", data)
	}
}

func TestRegisterServiceIdempotency(t *testing.T) {
	zookeeper := zookeeper()
	defer zookeeper.Client.Close()
	defer zookeeper.Client.Delete("/containerpilot/my-service/srv-id", -1)
	defer zookeeper.Client.Delete("/containerpilot/my-service", -1)
	defer zookeeper.Client.Delete("/containerpilot", -1)

	service := serviceDef("srv-id")
	err := zookeeper.registerService(service)
	err = zookeeper.registerService(service)
	if err != nil {
		t.Fatalf("RegisterService should be idempotent, %s", err)
	}
}

func TestDeregisterService(t *testing.T) {
	zookeeper := zookeeper()
	defer zookeeper.Client.Close()
	service := serviceDef("srv-id")
	zookeeper.registerService(service)
	zookeeper.Deregister(service)
	if err := zookeeper.Client.Delete("/containerpilot/my-service", -1); err != nil {
		t.Fatalf("Unable to cancel parent node: %s", err)
	}
	if err := zookeeper.Client.Delete("/containerpilot", -1); err != nil {
		t.Fatalf("Unable to cancel grand parent node: %s", err)
	}
}

func TestDeregisteServiceIdempotency(t *testing.T) {
	zookeeper := zookeeper()
	defer zookeeper.Client.Close()
	service := serviceDef("srv-id")
	zookeeper.registerService(service)
	zookeeper.Deregister(service)
	zookeeper.Deregister(service)
	if err := zookeeper.Client.Delete("/containerpilot/my-service", -1); err != nil {
		t.Fatalf("Unable to cancel parent node: %s", err)
	}
	if err := zookeeper.Client.Delete("/containerpilot", -1); err != nil {
		t.Fatalf("Unable to cancel grand parent node: %s", err)
	}
}

func TestMarkForMaintenanceService(t *testing.T) {
	zookeeper := zookeeper()
	defer zookeeper.Client.Close()
	service := serviceDef("srv-id")
	zookeeper.registerService(service)
	zookeeper.MarkForMaintenance(service)
	if err := zookeeper.Client.Delete("/containerpilot/my-service", -1); err != nil {
		t.Fatalf("Unable to cancel parent node: %s", err)
	}
	if err := zookeeper.Client.Delete("/containerpilot", -1); err != nil {
		t.Fatalf("Unable to cancel grand parent node: %s", err)
	}
}

func TestGetServices(t *testing.T) {
	zookeeper := zookeeper()
	defer zookeeper.Client.Close()
	defer zookeeper.Client.Delete("/containerpilot/my-service", -1)
	defer zookeeper.Client.Delete("/containerpilot", -1)
	service1 := serviceDef("srv-id-1")
	service2 := serviceDef("srv-id-2")
	service3 := serviceDef("srv-id-3")
	defer zookeeper.Deregister(service1)
	defer zookeeper.Deregister(service2)
	defer zookeeper.Deregister(service3)

	services, _ := zookeeper.getServices("my-service")
	if len(services) > 0 {
		t.Fatalf("services should be an empty array at this point %s", services)
	}
	zookeeper.registerService(service1)
	zookeeper.registerService(service2)
	zookeeper.registerService(service3)
	services, _ = zookeeper.getServices("my-service")
	if len(services) != 3 {
		t.Fatalf("now services should contain the three services: %s", services)
	}
	if services[0].ID != "srv-id-1" ||
		services[1].ID != "srv-id-2" ||
		services[2].ID != "srv-id-3" {
		t.Fatalf(
			"Unexpected IDs: %s, %s %s",
			services[0].ID,
			services[1].ID,
			services[2].ID,
		)
	}
}

func TestZookeeperCompareForChange(t *testing.T) {
	s1 := ServiceNode{
		ID:      "srv",
		Name:    "srv",
		Address: "192.168.1.1",
		Port:    9000,
	}
	if zookeeperCompareForChange([]ServiceNode{s1}, []ServiceNode{s1}) {
		t.Fatalf("The same object should return false")
	}
	s2 := ServiceNode{
		ID:      "srv2",
		Name:    "srv2",
		Address: "192.168.1.1",
		Port:    9000,
	}
	if zookeeperCompareForChange([]ServiceNode{s1}, []ServiceNode{s2}) {
		t.Fatalf("ID and name should not matter for comparison %s %s", s1, s2)
	}
	s2 = ServiceNode{
		ID:      "whatever",
		Name:    "whatever",
		Address: "192.168.1.2",
		Port:    9000,
	}
	if !zookeeperCompareForChange([]ServiceNode{s1}, []ServiceNode{s2}) {
		t.Fatalf("Address should matter for comparison %s %s", s1, s2)
	}
	s2 = ServiceNode{
		ID:      "whatever",
		Name:    "whatever",
		Address: "192.168.1.1",
		Port:    9001,
	}
	if !zookeeperCompareForChange([]ServiceNode{s1}, []ServiceNode{s2}) {
		t.Fatalf("Port should matter for comparison %s %s", s1, s2)
	}
}

func TestCheckForUpstreamChanges(t *testing.T) {
	zookeeper := zookeeper()
	defer zookeeper.Client.Delete("/containerpilot/my-service", -1)
	defer zookeeper.Client.Delete("/containerpilot", -1)
	service1 := serviceDef("srv-id-1")
	didChange := zookeeper.CheckForUpstreamChanges("my-service", "")
	if didChange {
		t.Fatalf("Should return false when no service is registered")
	}
	zookeeper.registerService(service1)
	defer zookeeper.Deregister(service1)

	didChange = zookeeper.CheckForUpstreamChanges("my-service", "")
	if !didChange {
		t.Fatalf("Should return true when a new service is registered")
	}
	didChange = zookeeper.CheckForUpstreamChanges("my-service", "")
	if didChange {
		t.Fatalf("Check should be idempotent")
	}
}

func TestZookeeperTTLPass(t *testing.T) {
	zookeeper := zookeeper()
	service := serviceDef("srv-id")
	defer zookeeper.Deregister(service)
	defer zookeeper.Client.Delete("/containerpilot/my-service", -1)
	defer zookeeper.Client.Delete("/containerpilot", -1)

	zookeeper.SendHeartbeat(service) // force registration
	_, _, err := zookeeper.Client.Get("/containerpilot/my-service/srv-id")
	if err != nil {
		t.Fatalf("Service is not registered, %s", err)
	}

	zookeeper.SendHeartbeat(service) // write TTL and verify
	_, _, err = zookeeper.Client.Get("/containerpilot/my-service/srv-id")
	if err != nil {
		t.Fatalf("Expected service to be registered, but was not, %s", err)
	}
	time.Sleep(2 * time.Second)

	_, _, err = zookeeper.Client.Get("/containerpilot/my-service/srv-id")
	if err == nil {
		t.Fatalf("Expected service to be deregistered, %s", err)
	}
}
