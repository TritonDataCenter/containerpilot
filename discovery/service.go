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
func (service *ServiceDefinition) SendHeartbeat() {
	if !service.wasRegistered {
		if err := service.registerService(); err != nil {
			log.Warnf("service registration failed: %s", err)
			return
		}
		service.wasRegistered = true
	}
	if err := service.Consul.PassTTL(service.ID, "ok"); err != nil {
		log.Infof("service not registered: %v", err)
		if err = service.registerService(); err != nil {
			log.Warnf("service registration failed: %s", err)
			return
		}
		if err = service.registerCheck(); err != nil {
			log.Warnf("check registration failed: %s", err)
			return
		}
		// now that we're ensured we're registered, we can push the
		// heartbeat again
		if err := service.Consul.PassTTL(service.ID, "ok"); err != nil {
			log.Errorf("Failed to write heartbeat: %s", err)
		}
		log.Infof("Service registered: %v", service.Name)
	}
}

func (service *ServiceDefinition) registerService() error {
	return service.Consul.ServiceRegister(
		&api.AgentServiceRegistration{
			ID:                service.ID,
			Name:              service.Name,
			Tags:              service.Tags,
			Port:              service.Port,
			Address:           service.IPAddress,
			EnableTagOverride: service.EnableTagOverride,
		},
	)
}

func (service *ServiceDefinition) registerCheck() error {
	return service.Consul.CheckRegister(
		&api.AgentCheckRegistration{
			ID:        service.ID,
			Name:      service.ID,
			Notes:     fmt.Sprintf("TTL for %s set by containerpilot", service.Name),
			ServiceID: service.ID,
			AgentServiceCheck: api.AgentServiceCheck{
				TTL: fmt.Sprintf("%ds", service.TTL),
				DeregisterCriticalServiceAfter: service.DeregisterCriticalServiceAfter,
			},
		},
	)
}
