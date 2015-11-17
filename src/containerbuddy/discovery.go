package main

type DiscoveryService interface {
	SendHeartbeat(*ServiceConfig)
	CheckForUpstreamChanges(*BackendConfig) bool
	MarkForMaintenance(*ServiceConfig)
}
