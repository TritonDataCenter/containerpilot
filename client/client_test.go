package client

import (
	"testing"
)

func TestNewHTTPClient(t *testing.T) {
	t.Skip("Not implemented")

	// Test NewHTTPClient. Test missing unix socket file
	// Test passing in nil socketPath....(? do we even need this ?)
}

func TestClientReload(t *testing.T) {
	t.Skip("Not implemented")

	// Bring up an HTTP server on a unix socket file
	// Setup route to our endpoint
	// Do not include proper endpoint
	// Return status 200
	// Return status 500
}

func TestClientSetMaintenance(t *testing.T) {
	t.Skip("Not implemented")

	// Test bool input switches endpoint to enable
	// Test bool input switches endpoint to disable
}

func TestClientPutEnv(t *testing.T) {
	t.Skip("Not implemented")

	// Test string input passes through Post call
}

func TestClientPutMetric(t *testing.T) {
	t.Skip("Not implemented")

	// Test string input passes through Post call
}
