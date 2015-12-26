package main

import (
	"fmt"
	"log"
	"sort"

	consul "github.com/hashicorp/consul/api"
)

// Consul is a service discovery backend for Hashicorp Consul
type Consul struct{ consul.Client }

// NewConsulConfig creates a new service discovery backend for Consul
func NewConsulConfig(uri string) Consul {
	client, _ := consul.NewClient(&consul.Config{
		Address: uri,
		Scheme:  "http",
	})
	config := &Consul{*client}
	return *config
}

// Deregister removes the node from Consul.
func (c Consul) Deregister(service *ServiceConfig) {
	c.MarkForMaintenance(service)
}

// MarkForMaintenance removes the node from Consul.
func (c Consul) MarkForMaintenance(service *ServiceConfig) {
	if err := c.Agent().ServiceDeregister(service.ID); err != nil {
		log.Printf("Deregistering failed: %s\n", err)
	}
}

// SendHeartbeat writes a TTL check status=ok to the consul store.
// If consul has never seen this service, we register the service and
// its TTL check.
func (c Consul) SendHeartbeat(service *ServiceConfig) {
	if err := c.Agent().PassTTL(service.ID, "ok"); err != nil {
		log.Printf("%v\nService not registered, registering...", err)
		if err = c.registerService(*service); err != nil {
			log.Printf("Service registration failed: %s\n", err)
		}
		if err = c.registerCheck(*service); err != nil {
			log.Printf("Check registration failed: %s\n", err)
		}
	}
}

func (c *Consul) registerService(service ServiceConfig) error {
	return c.Agent().ServiceRegister(
		&consul.AgentServiceRegistration{
			ID:      service.ID,
			Name:    service.Name,
			Port:    service.Port,
			Address: service.ipAddress,
		},
	)
}

func (c *Consul) registerCheck(service ServiceConfig) error {
	return c.Agent().CheckRegister(
		&consul.AgentCheckRegistration{
			ID:        service.ID,
			Name:      service.ID,
			Notes:     fmt.Sprintf("TTL for %s set by containerbuddy", service.Name),
			ServiceID: service.ID,
			AgentServiceCheck: consul.AgentServiceCheck{
				TTL: fmt.Sprintf("%ds", service.TTL),
			},
		},
	)
}

var upstreams = make(map[string][]*consul.ServiceEntry)

// CheckForUpstreamChanges runs the health check
func (c Consul) CheckForUpstreamChanges(backend *BackendConfig) bool {
	return c.checkHealth(*backend)
}

func (c *Consul) checkHealth(backend BackendConfig) bool {
	services, meta, err := c.Health().Service(backend.Name, "", true, nil)
	if err != nil {
		log.Printf("Failed to query %v: %s [%v]", backend.Name, err, meta)
		return false
	}
	didChange := compareForChange(upstreams[backend.Name], services)
	if didChange || len(services) == 0 {
		// We don't want to cause an onChange event the first time we read-in
		// but we do want to make sure we've written the key for this map
		upstreams[backend.Name] = services
	}
	return didChange
}

// Compare the two arrays to see if the address or port has changed
// or if we've added or removed entries.
func compareForChange(existing, new []*consul.ServiceEntry) (changed bool) {

	if len(existing) != len(new) {
		return true
	}

	sort.Sort(ByServiceID(existing))
	sort.Sort(ByServiceID(new))
	for i, ex := range existing {
		if ex.Service.Address != new[i].Service.Address ||
			ex.Service.Port != new[i].Service.Port {
			return true
		}
	}
	return false
}

// ByServiceID implements the Sort interface because Go can't sort without it.
type ByServiceID []*consul.ServiceEntry

func (se ByServiceID) Len() int           { return len(se) }
func (se ByServiceID) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByServiceID) Less(i, j int) bool { return se[i].Service.ID < se[j].Service.ID }
