package discovery

import (
	"fmt"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

// ServiceDefinition is how a job communicates with the Consul service
// discovery backend.
type ServiceDefinition struct {
	ID                             string
	Name                           string
	Port                           int
	TTL                            int
	Tags                           []string
	Meta                           map[string]string
	InitialStatus                  string
	IPAddress                      string
	EnableTagOverride              bool
	DeregisterCriticalServiceAfter string
	Consul                         Backend

	wasRegistered bool
}

// Deregister removes the service from Consul.
func (service *ServiceDefinition) Deregister() {
	log.Debugf("deregistering: %s", service.ID)
	if err := service.Consul.ServiceDeregister(service.ID); err != nil {
		log.Infof("deregistering failed: %s", err)
	}
}

// MarkForMaintenance removes the service from Consul.
func (service *ServiceDefinition) MarkForMaintenance() {
	service.Deregister()
}

// SendHeartbeat writes a TTL check status=ok to the Consul store.
func (service *ServiceDefinition) SendHeartbeat() error {
	// Make sure the service is registered.
	service.register(api.HealthPassing)

	checkID := fmt.Sprintf("service:%s", service.ID)
	if err := service.Consul.UpdateTTL(checkID, "ok", "pass"); err != nil {
		log.Warnf("service update TTL failed: %s", err)
	}

	return nil
}

// RegisterWithInitialStatus registers the service with its configured initial status.
func (service *ServiceDefinition) RegisterWithInitialStatus() {
	if service.wasRegistered {
		return
	}

	status := ""

	switch service.InitialStatus {
	case "passing":
		status = api.HealthPassing
	case "warning":
		status = api.HealthWarning
	case "critical":
		status = api.HealthCritical
	}

	log.Infof("Registering service %v with initial status set to %v",
		service.Name, service.InitialStatus)
	service.register(status)
}

// Register registers the service with the given status in Consul.
func (service *ServiceDefinition) register(status string) error {
	if !service.wasRegistered {
		if err := service.registerService(status); err != nil {
			log.Warnf("service registration failed: %s", err)
			return err
		}
		log.Infof("Service registered: %v", service.Name)
		service.wasRegistered = true
	}

	return nil
}

// registers the service along with a check set to the passing state
func (service *ServiceDefinition) registerService(status string) error {
	return service.Consul.ServiceRegister(
		&api.AgentServiceRegistration{
			ID:                service.ID,
			Name:              service.Name,
			Tags:              service.Tags,
			Meta:              service.Meta,
			Port:              service.Port,
			Address:           service.IPAddress,
			EnableTagOverride: service.EnableTagOverride,
			Check: &api.AgentServiceCheck{
				TTL:                            fmt.Sprintf("%ds", service.TTL),
				Status:                         status,
				Notes:                          fmt.Sprintf("TTL for %s set by containerpilot", service.Name),
				DeregisterCriticalServiceAfter: service.DeregisterCriticalServiceAfter,
			},
		},
	)
}
