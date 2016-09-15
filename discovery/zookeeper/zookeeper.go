package zookeeper

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
	"github.com/samuel/go-zookeeper/zk"
)

func init() {
	discovery.RegisterBackend("zookeeper", ConfigHook)
}

// ZooKeeper wraps a ZooKeeper connection handler. It also stores the
// prefix under which ContainerPilot nodes will be registered.
type ZooKeeper struct {
	Client    *zk.Conn
	Prefix    string
	eventChan chan zk.Event
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

func (conn ZooKeeper) ttlHandler(service *discovery.ServiceDefinition, key string) {
	for {
		select {
		case ev := <-conn.eventChan:
			_, _, _, err := conn.Client.GetW(ev.Path)
			if err != nil {
				log.Warning(err)
				conn.Deregister(service)
				return
			}
		case <-time.After(time.Duration(service.TTL) * time.Second):
			conn.Deregister(service)
			return
		}
	}
}

func (conn ZooKeeper) eventChanCallBack(ev zk.Event) {
	if ev.Type == zk.EventNodeDataChanged {
		go func() {
			conn.eventChan <- ev
		}()
	}
}

// ConfigHook is the hook to register with the ZooKeeper backend
func ConfigHook(raw interface{}) (discovery.ServiceBackend, error) {
	return NewZooKeeperConfig(raw)
}

func connection(addresses []string, cb zk.EventCallback) (*zk.Conn, error) {
	c, _, err := zk.Connect(addresses, time.Second, zk.WithEventCallback(cb))
	return c, err
}

// NewZooKeeperConfig creates a new service discovery backend for zookeeper
func NewZooKeeperConfig(raw interface{}) (*ZooKeeper, error) {
	zookeeper := &ZooKeeper{
		Prefix:    "/containerpilot",
		eventChan: make(chan zk.Event),
	}
	config := &struct {
		Address string `mapstructure:"address"`
	}{}
	if err := utils.DecodeRaw(raw, &config); err != nil {
		return nil, err
	}
	conn, err := connection([]string{config.Address}, zookeeper.eventChanCallBack)
	if err != nil {
		return nil, err
	}
	zookeeper.Client = conn
	return zookeeper, nil
}

// Deregister removes this instance from the registry
func (conn *ZooKeeper) Deregister(service *discovery.ServiceDefinition) {
	key := conn.getNodeKey(service)
	log.Warnf("Deregistering %s", key)
	if err := conn.Client.Delete(key, -1); err != nil {
		log.Errorf("Error on deregistering %s: %s", key, err)
	}
}

// MarkForMaintenance removes this instance from the registry
func (conn *ZooKeeper) MarkForMaintenance(service *discovery.ServiceDefinition) {
	conn.Deregister(service)
}

// SendHeartbeat refreshes the associated zookeeper node by
// re-registering it.
func (conn *ZooKeeper) SendHeartbeat(service *discovery.ServiceDefinition) {
	if err := conn.registerService(service); err != nil {
		log.Warnf("Error registering service %s: %s", service.Name, err)
	}
}

func (conn *ZooKeeper) parentPath(service *discovery.ServiceDefinition) string {
	return fmt.Sprintf("%s/%s", conn.Prefix, service.Name)
}

func (conn *ZooKeeper) getNodeKey(service *discovery.ServiceDefinition) string {
	return fmt.Sprintf("%s/%s", conn.parentPath(service), service.ID)
}

func (conn *ZooKeeper) getAppKey(appName string) string {
	return fmt.Sprintf("%s/%s", conn.Prefix, appName)
}

var zookeeperUpstreams = make(map[string][]ServiceNode)

// CheckForUpstreamChanges checks another zookeeper node for changes
func (conn *ZooKeeper) CheckForUpstreamChanges(backendName, backendTag string) bool {
	// TODO: is there a way to filter by tag in zookeeper?
	services, err := conn.getServices(backendName)
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

func (conn *ZooKeeper) getServices(appName string) ([]ServiceNode, error) {
	var services []ServiceNode

	key := conn.getAppKey(appName)
	children, _, err := conn.Client.Children(key)
	if err != nil {
		return services, err
	}
	for i := range children {
		path := fmt.Sprintf("%s/%s", key, children[i])
		data, _, err := conn.Client.Get(path)
		if err != nil {
			return services, err
		}
		srv, err := decodeZooKeeperNodeValue(data)
		if err != nil {
			return services, err
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

func (conn ZooKeeper) createParentPath(path string) error {
	pathElements := strings.Split(path, "/")[1:]
	sep := "/"
	newPath := ""
	for i := range pathElements {
		newPath = strings.Join([]string{newPath, sep, pathElements[i]}, "")
		if _, err := conn.Client.Create(
			newPath,
			nil,
			0,
			zk.WorldACL(zk.PermAll)); err != nil && err != zk.ErrNodeExists {
			return err
		}
	}
	return nil
}

func (conn ZooKeeper) registerService(service *discovery.ServiceDefinition) error {
	key := conn.getNodeKey(service)
	if err := conn.createParentPath(conn.parentPath(service)); err != nil {
		return err
	}
	value := encodeZooKeeperNodeValue(service)
	if _, err := conn.Client.Create(
		key,
		[]byte(value),
		zk.FlagEphemeral,
		zk.WorldACL(zk.PermAll)); err != nil && err != zk.ErrNodeExists {
		return err
	}
	// Set the watcher and trigger the call via `Set`
	_, _, _, err := conn.Client.GetW(key)
	if err != nil {
		return err
	}
	if _, err = conn.Client.Set(key, []byte(value), -1); err != nil {
		return err
	}
	go conn.ttlHandler(service, key)
	return nil
}

func encodeZooKeeperNodeValue(service *discovery.ServiceDefinition) []byte {
	node := &ServiceNode{
		ID:      service.ID,
		Name:    service.Name,
		Address: service.IPAddress,
		Port:    service.Port,
	}
	result, err := json.Marshal(&node)
	if err != nil {
		log.Warnf("Unable to encode service: %s", err)
		return nil
	}
	return result
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
