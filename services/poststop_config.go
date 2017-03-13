package services

import (
	"github.com/joyent/containerpilot/discovery"
)

// NewPostStopConfig ...
func NewPostStopConfig(raw interface{}, disc discovery.Backend) (*ServiceConfig, error) {
	// TODO!
	if raw == nil {
		return nil, nil
	}
	return &ServiceConfig{Name: "postStop"}, nil
}
