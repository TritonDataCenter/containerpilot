package services

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
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
	Timeout          string      `mapstructure:"timeout"`
	IPAddress        string
	ConsulConfig     *ConsulConfig `mapstructure:"consul"`
	healthCheckCmd   *commands.Command
	discoveryService discovery.ServiceBackend
	definition       *discovery.ServiceDefinition
}

// ConsulConfig handles additional Consul configuration.
type ConsulConfig struct {
	EnableTagOverride              bool   `mapstructure:"enableTagOverride"`
	DeregisterCriticalServiceAfter string `mapstructure:"deregisterCriticalServiceAfter"`
}

// NewServices new services from a raw config
func NewServices(raw []interface{}, disc discovery.ServiceBackend) ([]*Service, error) {
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

// NewService creates a new service
func NewService(name string, poll, port, ttl int, interfaces interface{},
	tags []string, consulConfig *ConsulConfig, disc discovery.ServiceBackend) (*Service, error) {
	service := &Service{
		Name:         name,
		Poll:         poll,
		Port:         port,
		TTL:          ttl,
		Interfaces:   interfaces,
		Tags:         tags,
		ConsulConfig: consulConfig,
	}
	if err := parseService(service, disc); err != nil {
		return nil, err
	}
	return service, nil
}

func parseService(s *Service, disc discovery.ServiceBackend) error {
	if err := utils.ValidateServiceName(s.Name); err != nil {
		return err
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
	if s.HealthCheckExec != nil {
		cmd, err := commands.NewCommand(s.HealthCheckExec, s.Timeout,
			log.Fields{"process": "health", "serviceName": s.Name, "serviceID": s.ID})
		if err != nil {
			return fmt.Errorf("Could not parse `health` in service %s: %s", s.Name, err)
		}
		cmd.Name = fmt.Sprintf("%s.health", s.Name)
		s.healthCheckCmd = cmd
	}

	interfaces, ifaceErr := utils.ToStringArray(s.Interfaces)
	if ifaceErr != nil {
		return ifaceErr
	}

	ipAddress, err := utils.GetIP(interfaces)
	if err != nil {
		return err
	}
	s.IPAddress = ipAddress

	var consulExtras *discovery.ConsulExtras
	if s.ConsulConfig != nil {

		if s.ConsulConfig.DeregisterCriticalServiceAfter != "" {
			if _, err := time.ParseDuration(s.ConsulConfig.DeregisterCriticalServiceAfter); err != nil {
				return fmt.Errorf("Could not parse consul `deregisterCriticalServiceAfter` in service %s: %s", s.Name, err)
			}
		}

		consulExtras = &discovery.ConsulExtras{
			DeregisterCriticalServiceAfter: s.ConsulConfig.DeregisterCriticalServiceAfter,
			EnableTagOverride:              s.ConsulConfig.EnableTagOverride,
		}
	}

	s.definition = &discovery.ServiceDefinition{
		ID:           s.ID,
		Name:         s.Name,
		Port:         s.Port,
		TTL:          s.TTL,
		Tags:         s.Tags,
		IPAddress:    s.IPAddress,
		ConsulExtras: consulExtras,
	}
	return nil
}

// PollTime implements Pollable for Service
// It returns the service's poll interval.
func (s Service) PollTime() time.Duration {
	return time.Duration(s.Poll) * time.Second
}

// PollAction implements Pollable for Service.
// So long as we don't get an error back from CheckHealth, we write a TTL
// health check to the discovery service.
func (s *Service) PollAction() {
	if err := s.CheckHealth(); err == nil {
		s.SendHeartbeat()
	}
}

// PollStop is a no-op
func (s *Service) PollStop() {}

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
func (s *Service) CheckHealth() error {

	// if we have a valid Service but there's no health check
	// set, assume it always passes (ex. telemetry service).
	if s.healthCheckCmd == nil {
		return nil
	}
	return commands.RunWithTimeout(s.healthCheckCmd)
}
