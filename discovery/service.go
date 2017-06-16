package discovery

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)

// ServiceDefinition is how a job communicates with the Consul service
// discovery backend.
type ServiceDefinition struct {
	ID                             string
	Name                           string
	Port                           int
	TTL                            int
	Tags                           []string
	IPAddress                      string
	EnableTagOverride              bool
	DeregisterCriticalServiceAfter string
	Consul                         Backend

	wasRegistered bool
}

// Deregister removes the service from Consul.
func (service *ServiceDefinition) Deregister() {
	service.MarkForMaintenance()
}

// MarkForMaintenance removes the service from Consul.
func (service *ServiceDefinition) MarkForMaintenance() {
	log.Debugf("deregistering: %s", service.ID)
	if err := service.Consul.ServiceDeregister(service.ID); err != nil {
		log.Infof("deregistering failed: %s", err)
	}
}

// SendHeartbeat writes a TTL check status=ok to the consul store.
// If consul has never seen this service, we register the service and
// its TTL check.
func (service *ServiceDefinition) SendHeartbeat() error {
	if !service.wasRegistered {
		if err := service.registerService(); err != nil {
			log.Warnf("service registration failed: %s", err)
			return err
		}
		service.wasRegistered = true
		return nil
	}
	checkID := fmt.Sprintf("service:%s", service.ID)
	if err := service.Consul.PassTTL(checkID, "ok"); err != nil {
		log.Infof("service not registered: %v", err)
		if err = service.registerService(); err != nil {
			log.Warnf("service registration failed: %s", err)
			return err
		}
		log.Infof("Service registered: %v", service.Name)
	}
	return nil
}

// registers the service along with a check set to the passing state
func (service *ServiceDefinition) registerService() error {
	return service.Consul.ServiceRegister(
		&api.AgentServiceRegistration{
			ID:                service.ID,
			Name:              service.Name,
			Tags:              service.Tags,
			Port:              service.Port,
			Address:           service.IPAddress,
			EnableTagOverride: service.EnableTagOverride,
			Check: &api.AgentServiceCheck{
				TTL:    fmt.Sprintf("%ds", service.TTL),
				Status: api.HealthPassing,
				Notes:  fmt.Sprintf("TTL for %s set by containerpilot", service.Name),
				DeregisterCriticalServiceAfter: service.DeregisterCriticalServiceAfter,
			},
		},
	)
}
