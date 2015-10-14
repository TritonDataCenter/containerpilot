package main

import (
	// consul "github.com/hashicorp/consul/api"
	"log"
)

type DiscoveryService interface {
	WriteHealthCheck()
	CheckForUpstreamChanges() bool
}

type Consul struct {
	Url              string
	ServiceName      string
	TTL              int
	UpstreamServices []string
	LastState        interface{}
}

func (c *Consul) WriteHealthCheck() {
	log.Printf("health check is ok!")
}

func (c *Consul) CheckForUpstreamChanges() bool {
	log.Printf("no change!")
	return false
}
