package discovery

import "github.com/hashicorp/consul/api"

// Backend is an interface which all service discovery backends must implement
type Backend interface {
	CheckForUpstreamChanges(backendName string, backendTag string) (bool, bool)
	CheckRegister(check *api.AgentCheckRegistration) error
	PassTTL(checkID, note string) error
	ServiceDeregister(serviceID string) error
	ServiceRegister(service *api.AgentServiceRegistration) error
}
