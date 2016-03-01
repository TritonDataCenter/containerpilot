package containerbuddy

import (
	"fmt"
	"sort"
	"time"

	"encoding/json"

	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
)

// Etcd is a service discovery backend for CoreOS etcd
type Etcd struct {
	Client client.Client
	API    client.KeysAPI
	Prefix string
}

// EtcdServiceNode is an instance of a service
type EtcdServiceNode struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Port    int      `json:"port"`
	Tags    []string `json:"tags,omitempty"`
}

// NewEtcdConfig creates a new service discovery backend for etcd
func NewEtcdConfig(config map[string]interface{}) Etcd {
	etcd := Etcd{
		Prefix: "/containerbuddy",
	}
	etcdConfig := client.Config{}
	switch endpoints := config["endpoints"].(type) {
	case string:
		etcdConfig.Endpoints = []string{endpoints}
	case []string:
		etcdConfig.Endpoints = endpoints
	default:
		log.Fatal("Must provide etcd endpoints")
	}

	prefix, ok := config["prefix"].(string)
	if ok {
		etcd.Prefix = prefix
	}

	etcdClient, err := client.New(etcdConfig)
	if err != nil {
		log.Fatal(err)
	}
	etcd.Client = etcdClient
	etcd.API = client.NewKeysAPI(etcdClient)
	return etcd
}

// Deregister removes this instance from the registry
func (c Etcd) Deregister(service *ServiceConfig) {
	c.deregisterService(service)
}

// MarkForMaintenance removes this instance from the registry
func (c Etcd) MarkForMaintenance(service *ServiceConfig) {
	c.deregisterService(service)
}

// SendHeartbeat refreshes the TTL of this associated etcd node
func (c Etcd) SendHeartbeat(service *ServiceConfig) {
	if err := c.updateServiceTTL(service); err != nil {
		log.Infof("Service not registered, registering...")
		if err := c.registerService(service); err != nil {
			log.Warnf("Error registering service %s: %s", service.Name, err)
		}
	}
}

// CheckForUpstreamChanges checks another etcd node for changes
func (c Etcd) CheckForUpstreamChanges(backend *BackendConfig) bool {
	return c.checkHealth(backend)
}

func (c Etcd) getNodeKey(service *ServiceConfig) string {
	return fmt.Sprintf("%s/%s/%s", c.Prefix, service.Name, service.ID)
}

func (c Etcd) getAppKey(appName string) string {
	return fmt.Sprintf("%s/%s", c.Prefix, appName)
}

var etcdUpstreams = make(map[string][]EtcdServiceNode)

func (c Etcd) checkHealth(backend *BackendConfig) bool {
	services, err := c.getServices(backend.Name)
	if err != nil {
		if _, ok := err.(client.Error); !ok {
			log.Warnf("Failed to query %v: %s", backend.Name, err)
		}
		return false
	}
	didChange := etcdCompareForChange(etcdUpstreams[backend.Name], services)
	if didChange || len(services) == 0 {
		// We don't want to cause an onChange event the first time we read-in
		// but we do want to make sure we've written the key for this map
		etcdUpstreams[backend.Name] = services
	}
	return didChange
}

func (c Etcd) checkServiceExists(service *ServiceConfig) bool {
	key := c.getNodeKey(service)
	if _, err := c.API.Get(context.Background(), key, nil); err != nil {
		if etcdErr, ok := err.(client.Error); ok {
			return etcdErr.Code != client.ErrorCodeKeyNotFound
		}
		log.Warnf("Unexpected etcd Error on key %s: %s", key, err)
	}
	return true
}

func (c Etcd) getServices(appName string) ([]EtcdServiceNode, error) {
	var services []EtcdServiceNode

	key := c.getAppKey(appName)
	resp, err := c.API.Get(context.Background(), key, &client.GetOptions{Recursive: true})
	if err != nil {
		if etcdErr, ok := err.(client.Error); ok {
			if etcdErr.Code == client.ErrorCodeKeyNotFound {
				return services, nil
			}
		}
		log.Errorf("Unable to get services: %s: %s", key, err)
		return services, err
	}
	if !resp.Node.Dir {
		log.Errorf("Etcd key %s is not a directory", key)
		return services, err
	}
	for _, instance := range resp.Node.Nodes {
		if !instance.Dir {
			continue
		}
		for _, node := range instance.Nodes {
			if service, err := decodeEtcdNodeValue(node); err != nil {
				log.Warnf("Could not decode etcd service %s: %s", node.Value, err)
			} else {
				services = append(services, service)
			}
		}
	}
	return services, nil
}

// Compare the two arrays to see if the address or port has changed
// or if we've added or removed entries.
func etcdCompareForChange(existing, new []EtcdServiceNode) (changed bool) {
	if len(existing) != len(new) {
		return true
	}

	sort.Sort(ByEtcdServiceID(existing))
	sort.Sort(ByEtcdServiceID(new))
	for i, ex := range existing {
		if ex.Address != new[i].Address ||
			ex.Port != new[i].Port {
			return true
		}
	}
	return false
}

func (c Etcd) registerService(service *ServiceConfig) error {
	key := c.getNodeKey(service)
	serviceKey := fmt.Sprintf("%s/%s", key, "/service")
	value := encodeEtcdNodeValue(service)
	ttl, _ := time.ParseDuration(fmt.Sprintf("%ds", service.TTL))
	// If the directory already exists, then this should silently fail (no error)
	if _, err := c.API.Set(context.Background(), key, "",
		&client.SetOptions{Dir: true, TTL: ttl, PrevExist: client.PrevIgnore}); err != nil {
		return err
	}
	// If the key exists, this should silently fail - no work to do, and don't want
	// to trigger any watches / updates
	_, err := c.API.Set(context.Background(), serviceKey, value,
		&client.SetOptions{PrevExist: client.PrevIgnore})
	return err
}

func (c Etcd) updateServiceTTL(service *ServiceConfig) error {
	key := c.getNodeKey(service)
	ttl, _ := time.ParseDuration(fmt.Sprintf("%ds", service.TTL))
	_, err := c.API.Set(context.Background(), key, "",
		&client.SetOptions{Dir: true, TTL: ttl, PrevExist: client.PrevExist})
	return err
}

func (c Etcd) deregisterService(service *ServiceConfig) error {
	_, err := c.API.Delete(context.Background(), c.getNodeKey(service),
		&client.DeleteOptions{Dir: true, Recursive: true})
	return err
}

func encodeEtcdNodeValue(service *ServiceConfig) string {
	node := &EtcdServiceNode{
		ID:      service.ID,
		Name:    service.Name,
		Address: service.ipAddress,
		Port:    service.Port,
	}
	json, err := json.Marshal(&node)
	if err != nil {
		log.Warnf("Unable to encode service: %s", err)
		return ""
	}
	return string(json)
}

func decodeEtcdNodeValue(node *client.Node) (EtcdServiceNode, error) {
	service := EtcdServiceNode{}
	err := json.Unmarshal([]byte(node.Value), &service)
	if err != nil {
		return service, err
	}
	return service, nil
}

// ByEtcdServiceID implements the Sort interface because Go can't sort without it.
type ByEtcdServiceID []EtcdServiceNode

func (se ByEtcdServiceID) Len() int           { return len(se) }
func (se ByEtcdServiceID) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByEtcdServiceID) Less(i, j int) bool { return se[i].ID < se[j].ID }
