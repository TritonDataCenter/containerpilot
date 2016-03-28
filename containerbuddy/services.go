package containerbuddy

import (
	"encoding/json"
	"os/exec"
	"utils"
)

// ServiceConfig configures the service, discovery data, and health checks
type ServiceConfig struct {
	ID               string
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	HealthCheckExec  json.RawMessage `json:"health"`
	Port             int             `json:"port"`
	TTL              int             `json:"ttl"`
	Interfaces       json.RawMessage `json:"interfaces"`
	Tags             []string        `json:"tags,omitempty"`
	discoveryService DiscoveryService
	ipAddress        string
	healthCheckCmd   *exec.Cmd
}

// PollTime implements Pollable for ServiceConfig
// It returns the service's poll interval.
func (s ServiceConfig) PollTime() int {
	return s.Poll
}

// PollAction implements Pollable for ServiceConfig.
// If the error code returned by CheckHealth is 0, we write a TTL health check
// to the discovery service.
func (s *ServiceConfig) PollAction() {
	if code, _ := s.CheckHealth(); code == 0 {
		s.SendHeartbeat()
	}
}

// SendHeartbeat sends a heartbeat for this service
func (s *ServiceConfig) SendHeartbeat() {
	s.discoveryService.SendHeartbeat(s)
}

// MarkForMaintenance marks this service for maintenance
func (s *ServiceConfig) MarkForMaintenance() {
	s.discoveryService.MarkForMaintenance(s)
}

// Deregister will deregister this instance of the service
func (s *ServiceConfig) Deregister() {
	s.discoveryService.Deregister(s)
}

// CheckHealth runs the service's health command, returning the results
func (s *ServiceConfig) CheckHealth() (int, error) {
	// if we have a valid ServiceConfig but there's no health check
	// set, assume it always passes (ex. metrics service).
	if s.healthCheckCmd == nil {
		return 0, nil
	}
	exitCode, err := run(s.healthCheckCmd)
	// Reset command object - since it can't be reused
	s.healthCheckCmd = utils.ArgsToCmd(s.healthCheckCmd.Args)
	return exitCode, err
}
