package etcd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"github.com/joyent/containerpilot/utils"
	"golang.org/x/net/context"

	"github.com/joyent/containerpilot/discovery"
)

func init() {
	discovery.RegisterBackend("etcd", ConfigHook)
}

// Etcd is a service discovery backend for CoreOS etcd
type Etcd struct {
	Client client.Client
	API    client.KeysAPI
	Prefix string
}

// ServiceNode is the serializable form of an Etcd service record
type ServiceNode struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Address string   `json:"address"`
	Port    int      `json:"port"`
	Tags    []string `json:"tags"`
}

type etcdRawConfig struct {
	Endpoints interface{} `mapstructure:"endpoints"`
	Prefix    string      `mapstructure:"prefix"`
}

func parseEndpoints(endpoints interface{}) []string {
	switch e := endpoints.(type) {
	case string:
		return []string{e}
	case []string:
		return e
	case []interface{}:
		var result []string
		for _, i := range e {
			if str, ok := i.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	log.Fatal("Must provide etcd endpoints")
	return nil
}

// ConfigHook is the hook to register with the Etcd backend
func ConfigHook(raw interface{}) (discovery.DiscoveryService, error) {
	return NewEtcdConfig(raw)
}

// NewEtcdConfig creates a new service discovery backend for etcd
func NewEtcdConfig(raw interface{}) (*Etcd, error) {
	etcd := &Etcd{
		Prefix: "/containerpilot",
	}
	var config etcdRawConfig
	etcdConfig := client.Config{}
	if err := utils.DecodeRaw(raw, &config); err != nil {
		return nil, err
	}
	etcdConfig.Endpoints = parseEndpoints(config.Endpoints)
	if config.Prefix != "" {
		etcd.Prefix = config.Prefix
	}

	etcdClient, err := client.New(etcdConfig)
	if err != nil {
		return nil, err
	}
	etcd.Client = etcdClient
	etcd.API = client.NewKeysAPI(etcdClient)
	return etcd, nil
}

// Deregister removes this instance from the registry
func (c *Etcd) Deregister(service *discovery.ServiceDefinition) {
	c.deregisterService(service)
}

// MarkForMaintenance removes this instance from the registry
func (c *Etcd) MarkForMaintenance(service *discovery.ServiceDefinition) {
	c.deregisterService(service)
}

// SendHeartbeat refreshes the TTL of this associated etcd node
func (c *Etcd) SendHeartbeat(service *discovery.ServiceDefinition) {
	if err := c.updateServiceTTL(service); err != nil {
		log.Infof("Service not registered, registering...")
		if err := c.registerService(service); err != nil {
			log.Warnf("Error registering service %s: %s", service.Name, err)
		}
	}
}

func (c *Etcd) getNodeKey(service *discovery.ServiceDefinition) string {
	return fmt.Sprintf("%s/%s/%s", c.Prefix, service.Name, service.ID)
}

func (c *Etcd) getAppKey(appName string) string {
	return fmt.Sprintf("%s/%s", c.Prefix, appName)
}

var etcdUpstreams = make(map[string][]ServiceNode)

// CheckForUpstreamChanges checks another etcd node for changes
func (c *Etcd) CheckForUpstreamChanges(backendName, backendTag string) bool {
	// TODO: is there a way to filter by tag in etcd?
	services, err := c.getServices(backendName)
	if err != nil {
		if _, ok := err.(client.Error); !ok {
			log.Warnf("Failed to query %v: %s", backendName, err)
		}
		return false
	}
	didChange := etcdCompareForChange(etcdUpstreams[backendName], services)
	if didChange || len(services) == 0 {
		// We don't want to cause an onChange event the first time we read-in
		// but we do want to make sure we've written the key for this map
		etcdUpstreams[backendName] = services
	}
	return didChange
}

func (c *Etcd) getServices(appName string) ([]ServiceNode, error) {
	var services []ServiceNode

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
func etcdCompareForChange(existing, new []ServiceNode) (changed bool) {
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

func (c Etcd) registerService(service *discovery.ServiceDefinition) error {
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

func (c Etcd) updateServiceTTL(service *discovery.ServiceDefinition) error {
	key := c.getNodeKey(service)
	ttl, _ := time.ParseDuration(fmt.Sprintf("%ds", service.TTL))
	_, err := c.API.Set(context.Background(), key, "",
		&client.SetOptions{Dir: true, TTL: ttl, PrevExist: client.PrevExist})
	return err
}

func (c Etcd) deregisterService(service *discovery.ServiceDefinition) error {
	_, err := c.API.Delete(context.Background(), c.getNodeKey(service),
		&client.DeleteOptions{Dir: true, Recursive: true})
	return err
}

func encodeEtcdNodeValue(service *discovery.ServiceDefinition) string {
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

func decodeEtcdNodeValue(node *client.Node) (ServiceNode, error) {
	service := ServiceNode{}
	err := json.Unmarshal([]byte(node.Value), &service)
	if err != nil {
		return service, err
	}
	return service, nil
}

// ByEtcdServiceID implements the Sort interface because Go can't sort without it.
type ByEtcdServiceID []ServiceNode

func (se ByEtcdServiceID) Len() int           { return len(se) }
func (se ByEtcdServiceID) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByEtcdServiceID) Less(i, j int) bool { return se[i].ID < se[j].ID }
