package zookeeper

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
	"github.com/samuel/go-zookeeper/zk"
	"sort"
	"strings"
	"time"

	"github.com/joyent/containerpilot/discovery"
)

func init() {
	discovery.RegisterBackend("zookeeper", ConfigHook)
}

// ZooKeeper is a wrapper ZooKeeper connection. It also stores the
// prefix under which ContainerPilot nodes will be registered.
type ZooKeeper struct {
	Connection *zk.Conn
	Prefix     string
}

// ServiceNode is the serializable form of a ZooKeeper service record
type ServiceNode struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Port    int      `json:"port"`
	Tags    []string `json:"tags"`
}

type zookeeperRawConfig struct {
	Endpoints interface{} `mapstructure:"endpoints"`
	Prefix    string      `mapstructure:"prefix"`
}

// ConfigHook is the hook to register with the ZooKeeper backend
func ConfigHook(raw interface{}) (discovery.ServiceBackend, error) {
	return NewZooKeeperConfig(raw)
}

// NewZooKeeperConfig creates a new service discovery backend for zookeeper
func NewZooKeeperConfig(raw interface{}) (*ZooKeeper, error) {
	zookeeper := &ZooKeeper{Prefix: "/containerpilot"}
	config := &struct {
		Address string `mapstructure:"address"`
	}{}
	if err := utils.DecodeRaw(raw, &config); err != nil {
		return nil, err
	}
	c, _, err := zk.Connect([]string{config.Address}, time.Second)
	if err != nil {
		return nil, err
	}
	zookeeper.Connection = c
	return zookeeper, nil
}

// Deregister removes this instance from the registry
func (c *ZooKeeper) Deregister(service *discovery.ServiceDefinition) {
	c.Connection.Delete(c.getNodeKey(service), -1)
}

// MarkForMaintenance removes this instance from the registry
func (c *ZooKeeper) MarkForMaintenance(service *discovery.ServiceDefinition) {
	c.Deregister(service)
}

// SendHeartbeat refreshes the TTL of this associated zookeeper node
func (c *ZooKeeper) SendHeartbeat(service *discovery.ServiceDefinition) {
	if err := c.registerService(service); err != nil {
		log.Warnf("Error registering service %s: %s", service.Name, err)
	}
}

func (c *ZooKeeper) parentPath(service *discovery.ServiceDefinition) string {
	return fmt.Sprintf("%s/%s", c.Prefix, service.Name)
}

func (c *ZooKeeper) getNodeKey(service *discovery.ServiceDefinition) string {
	return fmt.Sprintf("%s/%s", c.parentPath(service), service.ID)
}

func (c *ZooKeeper) getAppKey(appName string) string {
	return fmt.Sprintf("%s/%s", c.Prefix, appName)
}

var zookeeperUpstreams = make(map[string][]ServiceNode)

// CheckForUpstreamChanges checks another zookeeper node for changes
func (c *ZooKeeper) CheckForUpstreamChanges(backendName, backendTag string) bool {
	// TODO: is there a way to filter by tag in zookeeper?
	services, err := c.getServices(backendName)
	if err != nil {
		log.Errorf("Failed to query %v: %s", backendName, err)
		return false
	}
	didChange := zookeeperCompareForChange(zookeeperUpstreams[backendName], services)
	if didChange || len(services) == 0 {
		// We don't want to cause an onChange event the first time we read-in
		// but we do want to make sure we've written the key for this map
		zookeeperUpstreams[backendName] = services
	}
	return didChange
}

func (c *ZooKeeper) getServices(appName string) ([]ServiceNode, error) {
	var services []ServiceNode

	key := c.getAppKey(appName)
	children, _, error := c.Connection.Children(key)
	if error != nil {
		return services, error
	}
	for i := range children {
		path := fmt.Sprintf("%s/%s", key, children[i])
		data, _, error := c.Connection.Get(path)
		if error != nil {
			return services, error
		}
		srv, error := decodeZooKeeperNodeValue(data)
		if error != nil {
			return services, error
		}
		services = append(services, srv)
	}
	return services, nil
}

// Compare the two arrays to see if the address or port has changed
// or if we've added or removed entries.
func zookeeperCompareForChange(existing, new []ServiceNode) (changed bool) {
	if len(existing) != len(new) {
		return true
	}

	sort.Sort(ByZooKeeperServiceID(existing))
	sort.Sort(ByZooKeeperServiceID(new))
	for i, ex := range existing {
		if ex.Address != new[i].Address ||
			ex.Port != new[i].Port {
			return true
		}
	}
	return false
}

func (c ZooKeeper) createParentPath(path string) error {
	pathElements := strings.Split(path, "/")[1:]
	sep := "/"
	newPath := ""
	for i := range pathElements {
		newPath = strings.Join([]string{newPath, sep, pathElements[i]}, "")
		if exists, _, _ := c.Connection.Exists(newPath); !exists {
			if _, err := c.Connection.Create(newPath, nil, 0, zk.WorldACL(zk.PermAll)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c ZooKeeper) registerService(service *discovery.ServiceDefinition) error {
	k := c.getNodeKey(service)
	if err := c.createParentPath(c.parentPath(service)); err != nil {
		return err
	}
	value := encodeZooKeeperNodeValue(service)
	if exists, _, _ := c.Connection.Exists(k); !exists {
		if _, err := c.Connection.Create(k, []byte(value), 0, zk.WorldACL(zk.PermAll)); err != nil {
			return err
		}
		_, _, ch, _ := c.Connection.GetW(k)
		go func() {
			select {
			case ev := <-ch:
				_, _, ch, _ = c.Connection.GetW(ev.Path)
			case <-time.After(time.Duration(service.TTL) * time.Second):
				log.Warningf("TTL expired, deregistering %s", k)
				c.Deregister(service)
			}
		}()
	}
	return nil
}

func encodeZooKeeperNodeValue(service *discovery.ServiceDefinition) string {
	node := &ServiceNode{
		ID:      service.ID,
		Name:    service.Name,
		Address: service.IPAddress,
		Port:    service.Port,
	}
	json, err := json.Marshal(&node)
	if err != nil {
		log.Warnf("Unable to encode service: %s", err)
		return ""
	}
	return string(json)
}

func decodeZooKeeperNodeValue(rawValue []byte) (ServiceNode, error) {
	service := ServiceNode{}
	err := json.Unmarshal(rawValue, &service)
	if err != nil {
		return service, err
	}
	return service, nil
}

// ByZooKeeperServiceID implements the Sort interface because Go can't sort without it.
type ByZooKeeperServiceID []ServiceNode

func (se ByZooKeeperServiceID) Len() int           { return len(se) }
func (se ByZooKeeperServiceID) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByZooKeeperServiceID) Less(i, j int) bool { return se[i].ID < se[j].ID }
