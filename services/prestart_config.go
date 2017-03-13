package services

import (
	"github.com/joyent/containerpilot/discovery"
)

// NewPreStartConfig ...
func NewPreStartConfig(raw interface{}, disc discovery.Backend) (*ServiceConfig, error) {
	// TODO!
	if raw == nil {
		return nil, nil
	}
	return &ServiceConfig{Name: "preStart"}, nil
}
