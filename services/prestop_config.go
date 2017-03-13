package services

import (
	"github.com/joyent/containerpilot/discovery"
)

// NewPreStopConfig ...
func NewPreStopConfig(raw interface{}, disc discovery.Backend) (*ServiceConfig, error) {
	// TODO!
	if raw == nil {
		return nil, nil
	}
	return &ServiceConfig{Name: "preStop"}, nil
}
