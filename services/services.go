package services

import (
	"discovery"
	"encoding/json"
	"fmt"
	"os"
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
	ipAddress        string
	healthCheckCmd   *exec.Cmd
	discoveryService discovery.DiscoveryService
	definition       *discovery.ServiceDefinition
}

func (s *ServiceConfig) Parse(discoveryService discovery.DiscoveryService) error {

	if s.Name == "" {
		return fmt.Errorf("service must have a `name`")
	}
	hostname, _ := os.Hostname()
	s.ID = fmt.Sprintf("%s-%s", s.Name, hostname)
	s.discoveryService = discoveryService
	if s.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in service %s", s.Name)
	}
	if s.TTL < 1 {
		return fmt.Errorf("`ttl` must be > 0 in service %s", s.Name)
	}
	if s.Port < 1 {
		return fmt.Errorf("`port` must be > 0 in service %s", s.Name)
	}

	if cmd, err := utils.ParseCommandArgs(s.HealthCheckExec); err != nil {
		return fmt.Errorf("Could not parse `health` in service %s: %s", s.Name, err)
	} else if cmd == nil {
		return fmt.Errorf("`health` is required in service %s", s.Name)
	} else {
		s.healthCheckCmd = cmd
	}

	interfaces, ifaceErr := utils.ParseInterfaces(s.Interfaces)
	if ifaceErr != nil {
		return ifaceErr
	}

	if ipAddress, err := utils.GetIP(interfaces); err != nil {
		return err
	} else {
		s.ipAddress = ipAddress
	}
	s.definition = &discovery.ServiceDefinition{
		ID:        s.ID,
		Name:      s.Name,
		Port:      s.Port,
		TTL:       s.TTL,
		Tags:      s.Tags,
		IpAddress: s.ipAddress,
	}
	return nil
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
	s.discoveryService.SendHeartbeat(s.definition)
}

// MarkForMaintenance marks this service for maintenance
func (s *ServiceConfig) MarkForMaintenance() {
	s.discoveryService.MarkForMaintenance(s.definition)
}

// Deregister will deregister this instance of the service
func (s *ServiceConfig) Deregister() {
	s.discoveryService.Deregister(s.definition)
}

// CheckHealth runs the service's health command, returning the results
func (s *ServiceConfig) CheckHealth() (int, error) {
	exitCode, err := utils.Run(s.healthCheckCmd)
	// Reset command object - since it can't be reused
	s.healthCheckCmd = utils.ArgsToCmd(s.healthCheckCmd.Args)
	return exitCode, err
}
