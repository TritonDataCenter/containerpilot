package main

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"log"
	"sort"
)

type DiscoveryService interface {
	WriteHealthCheck()
	CheckForUpstreamChanges() bool
}

type Consul struct {
	client           *consul.Client
	Address          string
	Ports            []int
	ServiceName      string
	ServiceId        string
	TTL              int
	UpstreamServices []string
	LastState        interface{}
}

func NewConsulConfig(uri, serviceId, serviceName, address string, ports []int, ttl int, toCheck []string) Consul {
	client, _ := consul.NewClient(&consul.Config{
		Address: uri,
		Scheme:  "http",
	})
	config := &Consul{
		client:           client,
		Address:          address,
		Ports:            ports,
		ServiceName:      serviceName,
		ServiceId:        serviceId,
		TTL:              ttl,
		UpstreamServices: toCheck,
	}
	return *config
}

// WriteHealthCheck writes a TTL check status=ok to the consul store.
// If consul has never seen this service, we register the service and
// its TTL check.
func (c Consul) WriteHealthCheck() {
	if err := c.client.Agent().PassTTL(c.ServiceId, "ok"); err != nil {
		log.Printf("%v\nService not registered, registering...", err)
		if err = c.registerService(); err != nil {
			log.Printf("Service registration failed: %s\n", err)
		}
		if err = c.registerCheck(); err != nil {
			log.Printf("Check registration failed: %s\n", err)
		}
	}
}

func (c *Consul) registerService() error {
	return c.client.Agent().ServiceRegister(
		&consul.AgentServiceRegistration{
			ID:      c.ServiceId,
			Name:    c.ServiceName,
			Port:    c.Ports[0], // TODO: need to support multiple ports
			Address: c.Address,
		},
	)
}

func (c *Consul) registerCheck() error {
	return c.client.Agent().CheckRegister(
		&consul.AgentCheckRegistration{
			ID:        c.ServiceId,
			Name:      c.ServiceId,
			Notes:     "???",
			ServiceID: c.ServiceId,
			AgentServiceCheck: consul.AgentServiceCheck{
				TTL: fmt.Sprintf("%ds", c.TTL),
			},
		},
	)
}

var upstreams = make(map[string][]*consul.ServiceEntry)

func (c Consul) CheckForUpstreamChanges() bool {
	for _, service := range c.UpstreamServices {
		if c.checkHealth(service) {
			return true
		}
	}
	return false
}

func (c *Consul) checkHealth(upstream string) bool {
	if services, meta, err := c.client.Health().Service(upstream, "", true, nil); err != nil {
		log.Printf("Failed to query %v: %s [%v]", upstream, err, meta)
		return false
	} else {
		didChange := compareForChange(upstreams[c.ServiceId], services)
		if didChange || len(services) == 0 {
			// We don't want to cause an onChange event the first time we read-in
			// but we do want to make sure we've written the key for this map
			upstreams[c.ServiceId] = services
		}
		return didChange
	}
}

// Compare the two arrays to see if the address or port has changed
// or if we've added or removed entries.
func compareForChange(existing, new []*consul.ServiceEntry) (changed bool) {

	if len(existing) != len(new) {
		return true
	}

	sort.Sort(ByServiceId(existing))
	sort.Sort(ByServiceId(new))
	for i, ex := range existing {
		if ex.Service.Address != new[i].Service.Address ||
			ex.Service.Port != new[i].Service.Port {
			return true
		}
	}
	return false
}

// Implement the Sort interface because Go can't sort without it.
type ByServiceId []*consul.ServiceEntry

func (se ByServiceId) Len() int           { return len(se) }
func (se ByServiceId) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByServiceId) Less(i, j int) bool { return se[i].Service.ID < se[j].Service.ID }
