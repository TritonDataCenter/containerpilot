package mocks

import "github.com/joyent/containerpilot/discovery"

// NoopDiscoveryBackend is a mock discovery.Backend
type NoopDiscoveryBackend struct {
	Val bool
}

// SendHeartbeat (required for mock interface)
func (noop *NoopDiscoveryBackend) SendHeartbeat(service *discovery.ServiceDefinition) {
	return
}

// CheckForUpstreamChanges will return the public Val field to mock
// whether a change has occurred.
func (noop *NoopDiscoveryBackend) CheckForUpstreamChanges(backend, tag string) bool {
	return noop.Val
}

// MarkForMaintenance (required for mock interface)
func (noop *NoopDiscoveryBackend) MarkForMaintenance(service *discovery.ServiceDefinition) {}

// Deregister (required for mock interface)
func (noop *NoopDiscoveryBackend) Deregister(service *discovery.ServiceDefinition) {}
