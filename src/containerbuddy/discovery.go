package main

type DiscoveryService interface {
	WriteHealthCheck(*ServiceConfig)
	CheckForUpstreamChanges(*BackendConfig) bool
}

