package main

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"log"
	"os"
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

func NewConsulConfig(uri, serviceName, address string, ports []int, ttl int, toCheck []string) Consul {
	client, _ := consul.NewClient(&consul.Config{
		Address: uri,
		Scheme:  "http",
	})
	hostname, _ := os.Hostname()
	config := &Consul{
		client:           client,
		Address:          address,
		Ports:            ports,
		ServiceName:      serviceName,
		ServiceId:        fmt.Sprintf("%s-%s", serviceName, hostname),
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

func (c Consul) CheckForUpstreamChanges() bool {
	log.Printf("no change!")
	return false
}
