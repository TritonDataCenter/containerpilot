package discovery

// DiscoveryService is an interface
// which all service discovery backends must implement
type DiscoveryService interface {
	SendHeartbeat(service *ServiceDefinition)
	CheckForUpstreamChanges(backendName string, backendTag string) bool
	MarkForMaintenance(service *ServiceDefinition)
	Deregister(service *ServiceDefinition)
}

type ServiceDefinition struct {
	ID        string
	Name      string
	Port      int
	TTL       int
	Tags      []string
	IpAddress string
}
