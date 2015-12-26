package main

// DiscoveryService is an interface
// which all service discovery backends must implement
type DiscoveryService interface {
	SendHeartbeat(*ServiceConfig)
	CheckForUpstreamChanges(*BackendConfig) bool
	MarkForMaintenance(*ServiceConfig)
	Deregister(*ServiceConfig)
}
