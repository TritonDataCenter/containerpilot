package main

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"log"
)

type DiscoveryService interface {
	WriteHealthCheck()
	CheckForUpstreamChanges() bool
}

type Consul struct {
	client           *consul.Client
	ServiceName      string
	TTL              int
	UpstreamServices []string
	LastState        interface{}
}

func NewConsulConfig(uri, serviceName string, ttl int, toCheck []string) *Consul {
	client, _ := consul.NewClient(&consul.Config{
		Address: uri,
		Scheme:  "http",
	})
	config := &Consul{
		client:           client,
		ServiceName:      serviceName,
		TTL:              ttl,
		UpstreamServices: toCheck,
	}
	return config
}

// WriteHealthCheck writes a TTL check status=ok to the consul store.
// If consul has never seen this service, we register the service and
// its TTL check.
func (c *Consul) WriteHealthCheck() {
	if err := c.client.Agent().PassTTL(c.ServiceName, "ok"); err != nil {
		log.Println("Service not registered, registering...")
		if err = c.client.Agent().ServiceRegister(
			&consul.AgentServiceRegistration{
				ID:      c.ServiceName, // TODO: name vs ID???
				Name:    c.ServiceName, // TODO: name vs ID???
				Port:    0,             // TODO: need to get address:port from config
				Address: "127.0.0.1",   // TODO: need to get address:port from config
				Check: &consul.AgentServiceCheck{
					TTL: fmt.Sprintf("%ds", c.TTL),
				},
			},
		); err != nil {
			log.Printf("PassTTL call failed: %s", err)
		}
	}
}

func (c *Consul) CheckForUpstreamChanges() bool {
	log.Printf("no change!")
	return false
}
