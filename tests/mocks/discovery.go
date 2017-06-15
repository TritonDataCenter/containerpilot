package mocks

import "github.com/hashicorp/consul/api"

// NoopDiscoveryBackend is a mock discovery.Backend
type NoopDiscoveryBackend struct {
	Val     bool
	lastVal bool
}

// CheckForUpstreamChanges will return the public Val field to mock
// whether a change has occurred. Will not report a change on the second
// check unless the Val has been updated externally by the test rig
func (noop *NoopDiscoveryBackend) CheckForUpstreamChanges(_, _, _ string) (didChange, isHealthy bool) {
	if noop.lastVal != noop.Val {
		didChange = true
	}
	noop.lastVal = noop.Val
	isHealthy = noop.Val
	return didChange, isHealthy
}

// CheckRegister (required for mock interface)
func (noop *NoopDiscoveryBackend) CheckRegister(check *api.AgentCheckRegistration) error {
	return nil
}

// PassTTL (required for mock interface)
func (noop *NoopDiscoveryBackend) PassTTL(checkID, note string) error {
	return nil
}

// ServiceDeregister (required for mock interface)
func (noop *NoopDiscoveryBackend) ServiceDeregister(serviceID string) error {
	return nil
}

// ServiceRegister (required for mock interface)
func (noop *NoopDiscoveryBackend) ServiceRegister(service *api.AgentServiceRegistration) error {
	return nil
}
