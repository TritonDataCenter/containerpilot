package mocks

import "github.com/joyent/containerpilot/discovery"

// NoopDiscoveryBackend is a mock discovery.Backend
type NoopDiscoveryBackend struct {
	Val     bool
	lastVal bool
}

// SendHeartbeat (required for mock interface)
func (noop *NoopDiscoveryBackend) SendHeartbeat(service *discovery.ServiceDefinition) {
	return
}

// CheckForUpstreamChanges will return the public Val field to mock
// whether a change has occurred. Will not report a change on the second
// check unless the Val has been updated externally by the test rig
func (noop *NoopDiscoveryBackend) CheckForUpstreamChanges(backend, tag string) (didChange, isHealthy bool) {
	if noop.lastVal != noop.Val {
		didChange = true
	}
	noop.lastVal = noop.Val
	isHealthy = noop.Val
	return didChange, isHealthy
}

// MarkForMaintenance (required for mock interface)
func (noop *NoopDiscoveryBackend) MarkForMaintenance(service *discovery.ServiceDefinition) {}

// Deregister (required for mock interface)
func (noop *NoopDiscoveryBackend) Deregister(service *discovery.ServiceDefinition) {}
