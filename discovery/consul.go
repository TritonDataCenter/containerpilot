package discovery

import (
	"fmt"
	"os"
	"sort"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

// Consul wraps the service discovery backend for the Hashicorp Consul client
// and tracks the state of all watched dependencies.
type Consul struct {
	api.Client
	lock            sync.RWMutex
	watchedServices map[string][]*api.ServiceEntry
}

// NewConsul creates a new service discovery backend for Consul
func NewConsul(config interface{}) (*Consul, error) {
	var consulConfig *api.Config
	var err error
	switch t := config.(type) {
	case string:
		consulConfig, err = configFromURI(t)
	case map[string]interface{}:
		consulConfig, err = configFromMap(t)
	default:
		return nil, fmt.Errorf("no discovery backend defined")
	}
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("CONSUL_HTTP_TOKEN"); token != "" {
		consulConfig.Token = token
	}
	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	watchedServices := make(map[string][]*api.ServiceEntry)
	consul := &Consul{*client, sync.RWMutex{}, watchedServices}
	return consul, nil
}

// PassTTL wraps the Consul.Agent's PassTTL method, and is used to set a
// TTL check to the passing state
func (c *Consul) PassTTL(name, note string) error {
	return c.Agent().PassTTL(name, note)
}

// CheckRegister wraps the Consul.Agent's CheckRegister method,
// is used to register a new service with the local agent
func (c *Consul) CheckRegister(check *api.AgentCheckRegistration) error {
	return c.Agent().CheckRegister(check)
}

// ServiceRegister wraps the Consul.Agent's ServiceRegister method,
// is used to register a new service with the local agent
func (c *Consul) ServiceRegister(service *api.AgentServiceRegistration) error {
	return c.Agent().ServiceRegister(service)
}

// ServiceDeregister wraps the Consul.Agent's ServiceDeregister method,
// and is used to deregister a service from the local agent
func (c *Consul) ServiceDeregister(serviceID string) error {
	return c.Agent().ServiceDeregister(serviceID)
}

// CheckForUpstreamChanges requests the set of healthy instances of a
// service from Consul and checks whether there has been a change since
// the last check.
func (c *Consul) CheckForUpstreamChanges(backendName, backendTag, dc string) (didChange, isHealthy bool) {
	opts := &api.QueryOptions{Datacenter: dc}
	instances, meta, err := c.Health().Service(backendName, backendTag, true, opts)
	if err != nil {
		log.Warnf("failed to query %v: %s [%v]", backendName, err, meta)
		return false, false
	}
	isHealthy = len(instances) > 0
	didChange = c.compareAndSwap(backendName, instances)
	return didChange, isHealthy
}

// returns true if any addresses for the service changed and updates
// the internal state
func (c *Consul) compareAndSwap(service string, new []*api.ServiceEntry) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	existing := c.watchedServices[service]
	c.watchedServices[service] = new
	return compareForChange(existing, new)
}

// Compare the two arrays to see if the address or port has changed
// or if we've added or removed entries.
func compareForChange(existing, newEntries []*api.ServiceEntry) (changed bool) {
	if len(existing) != len(newEntries) {
		return true
	}
	sort.Sort(ByServiceID(existing))
	sort.Sort(ByServiceID(newEntries))
	for i, ex := range existing {
		if ex.Service.Address != newEntries[i].Service.Address ||
			ex.Service.Port != newEntries[i].Service.Port {
			return true
		}
	}
	return false
}

// ByServiceID implements the Sort interface because Go can't sort without it.
type ByServiceID []*api.ServiceEntry

func (se ByServiceID) Len() int           { return len(se) }
func (se ByServiceID) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByServiceID) Less(i, j int) bool { return se[i].Service.ID < se[j].Service.ID }
