package services

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// Service configures the service, discovery data, and health checks
type Service struct {
	ID               string
	Name             string      `mapstructure:"name"`
	Poll             int         `mapstructure:"poll"` // time in seconds
	HealthCheckExec  interface{} `mapstructure:"health"`
	Port             int         `mapstructure:"port"`
	TTL              int         `mapstructure:"ttl"`
	Interfaces       interface{} `mapstructure:"interfaces"`
	Tags             []string    `mapstructure:"tags"`
	ipAddress        string
	healthCheckCmd   *exec.Cmd
	discoveryService discovery.DiscoveryService
	definition       *discovery.ServiceDefinition
}

func NewServices(raw []interface{}, disc discovery.DiscoveryService) ([]*Service, error) {
	if raw == nil {
		return []*Service{}, nil
	}
	var services []*Service
	if err := utils.DecodeRaw(raw, &services); err != nil {
		return nil, fmt.Errorf("Service configuration error: %v", err)
	}
	for _, s := range services {
		if err := parseService(s, disc); err != nil {
			return nil, err
		}
	}
	return services, nil
}

func NewService(name string, poll, port, ttl int, interfaces interface{},
	tags []string, disc discovery.DiscoveryService) (*Service, error) {
	service := &Service{
		Name:       name,
		Poll:       poll,
		Port:       port,
		TTL:        ttl,
		Interfaces: interfaces,
		Tags:       tags,
	}
	if err := parseService(service, disc); err != nil {
		return nil, err
	}
	return service, nil
}

func parseService(s *Service, disc discovery.DiscoveryService) error {
	if s.Name == "" {
		return fmt.Errorf("service must have a `name`")
	}
	hostname, _ := os.Hostname()
	s.ID = fmt.Sprintf("%s-%s", s.Name, hostname)
	s.discoveryService = disc
	if s.Poll < 1 {
		return fmt.Errorf("`poll` must be > 0 in service %s", s.Name)
	}
	if s.TTL < 1 {
		return fmt.Errorf("`ttl` must be > 0 in service %s", s.Name)
	}
	if s.Port < 1 {
		return fmt.Errorf("`port` must be > 0 in service %s", s.Name)
	}

	// if the HealthCheckExec is nil then we'll have no health check
	// command; this is useful for the telemetry service
	if cmd, err := utils.ParseCommandArgs(s.HealthCheckExec); err != nil {
		return fmt.Errorf("Could not parse `health` in service %s: %s", s.Name, err)
	} else {
		s.healthCheckCmd = cmd
	}

	interfaces, ifaceErr := utils.ToStringArray(s.Interfaces)
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

// PollTime implements Pollable for Service
// It returns the service's poll interval.
func (s Service) PollTime() time.Duration {
	return time.Duration(s.Poll) * time.Second
}

// PollAction implements Pollable for Service.
// If the error code returned by CheckHealth is 0, we write a TTL health check
// to the discovery service.
func (s *Service) PollAction() {
	if code, _ := s.CheckHealth(); code == 0 {
		s.SendHeartbeat()
	}
}

// PollStop does nothing in a Service
func (s *Service) PollStop() {
	// Nothing to do
}

// SendHeartbeat sends a heartbeat for this service
func (s *Service) SendHeartbeat() {
	s.discoveryService.SendHeartbeat(s.definition)
}

// MarkForMaintenance marks this service for maintenance
func (s *Service) MarkForMaintenance() {
	s.discoveryService.MarkForMaintenance(s.definition)
}

// Deregister will deregister this instance of the service
func (s *Service) Deregister() {
	s.discoveryService.Deregister(s.definition)
}

// CheckHealth runs the service's health command, returning the results
func (s *Service) CheckHealth() (int, error) {

	defer func() {
		// reset command object because it can't be reused
		if s.healthCheckCmd != nil {
			s.healthCheckCmd = utils.ArgsToCmd(s.healthCheckCmd.Args)
		}
	}()

	// if we have a valid Service but there's no health check
	// set, assume it always passes (ex. telemetry service).
	if s.healthCheckCmd == nil {
		return 0, nil
	}
	exitCode, err := utils.RunWithFields(s.healthCheckCmd, log.Fields{"process": "health", "serviceName": s.Name, "serviceID": s.ID})
	return exitCode, err
}
