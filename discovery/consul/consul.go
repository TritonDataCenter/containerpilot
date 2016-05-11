package consul

import (
	"fmt"
	"os"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

func init() {
	discovery.RegisterBackend("consul", ConfigHook)
}

// Consul is a service discovery backend for Hashicorp Consul
type Consul struct{ consul.Client }

// ConfigHook is the hook to register with the Consul backend
func ConfigHook(raw interface{}) (discovery.ServiceBackend, error) {
	return NewConsulConfig(raw)
}

// NewConsulConfig creates a new service discovery backend for Consul
func NewConsulConfig(config interface{}) (*Consul, error) {
	var consulConfig *consul.Config
	var err error
	switch t := config.(type) {
	case string:
		consulConfig, err = configFromURI(t)
	case map[string]interface{}:
		consulConfig, err = configFromMap(t)
	default:
		return nil, fmt.Errorf("Unexpected Consul config structure. Expected a string or map")
	}
	if err != nil {
		return nil, err
	}

	if token := os.Getenv("CONSUL_HTTP_TOKEN"); token != "" {
		consulConfig.Token = token
	}
	client, err := consul.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	return &Consul{*client}, nil
}

func configFromMap(raw map[string]interface{}) (*consul.Config, error) {
	config := &struct {
		Address string `mapstructure:"address"`
		Scheme  string `mapstructure:"scheme"`
		Token   string `mapstructure:"token"`
	}{}
	if err := utils.DecodeRaw(raw, config); err != nil {
		return nil, err
	}
	return &consul.Config{
		Address: config.Address,
		Scheme:  config.Scheme,
		Token:   config.Token,
	}, nil
}

func configFromURI(uri string) (*consul.Config, error) {
	address, scheme := parseRawURI(uri)
	return &consul.Config{
		Address: address,
		Scheme:  scheme,
	}, nil
}

// Returns the uri broken into an address and scheme portion
func parseRawURI(raw string) (string, string) {

	var scheme = "http" // default
	var address = raw   // we accept bare address w/o a scheme

	// strip the scheme from the prefix and (maybe) set the scheme to https
	if strings.HasPrefix(raw, "http://") {
		address = strings.Replace(raw, "http://", "", 1)
	} else if strings.HasPrefix(raw, "https://") {
		address = strings.Replace(raw, "https://", "", 1)
		scheme = "https"
	}
	return address, scheme
}

// Deregister removes the node from Consul.
func (c *Consul) Deregister(service *discovery.ServiceDefinition) {
	c.MarkForMaintenance(service)
}

// MarkForMaintenance removes the node from Consul.
func (c *Consul) MarkForMaintenance(service *discovery.ServiceDefinition) {
	if err := c.Agent().ServiceDeregister(service.ID); err != nil {
		log.Infof("Deregistering failed: %s", err)
	}
}

// SendHeartbeat writes a TTL check status=ok to the consul store.
// If consul has never seen this service, we register the service and
// its TTL check.
func (c *Consul) SendHeartbeat(service *discovery.ServiceDefinition) {
	if err := c.Agent().PassTTL(service.ID, "ok"); err != nil {
		log.Infof("%v\nService not registered, registering...", err)
		if err = c.registerService(*service); err != nil {
			log.Warnf("Service registration failed: %s", err)
		}
		if err = c.registerCheck(*service); err != nil {
			log.Warnf("Check registration failed: %s", err)
		}
	}
}

func (c *Consul) registerService(service discovery.ServiceDefinition) error {
	return c.Agent().ServiceRegister(
		&consul.AgentServiceRegistration{
			ID:      service.ID,
			Name:    service.Name,
			Tags:    service.Tags,
			Port:    service.Port,
			Address: service.IPAddress,
		},
	)
}

func (c *Consul) registerCheck(service discovery.ServiceDefinition) error {
	return c.Agent().CheckRegister(
		&consul.AgentCheckRegistration{
			ID:        service.ID,
			Name:      service.ID,
			Notes:     fmt.Sprintf("TTL for %s set by containerpilot", service.Name),
			ServiceID: service.ID,
			AgentServiceCheck: consul.AgentServiceCheck{
				TTL: fmt.Sprintf("%ds", service.TTL),
			},
		},
	)
}

var upstreams = make(map[string][]*consul.ServiceEntry)

// CheckForUpstreamChanges runs the health check
func (c Consul) CheckForUpstreamChanges(backendName, backendTag string) bool {
	services, meta, err := c.Health().Service(backendName, backendTag, true, nil)
	if err != nil {
		log.Warnf("Failed to query %v: %s [%v]", backendName, err, meta)
		return false
	}
	didChange := compareForChange(upstreams[backendName], services)
	if didChange || len(services) == 0 {
		// We don't want to cause an onChange event the first time we read-in
		// but we do want to make sure we've written the key for this map
		upstreams[backendName] = services
	}
	return didChange
}

// Compare the two arrays to see if the address or port has changed
// or if we've added or removed entries.
func compareForChange(existing, new []*consul.ServiceEntry) (changed bool) {

	if len(existing) != len(new) {
		return true
	}

	sort.Sort(ByServiceID(existing))
	sort.Sort(ByServiceID(new))
	for i, ex := range existing {
		if ex.Service.Address != new[i].Service.Address ||
			ex.Service.Port != new[i].Service.Port {
			return true
		}
	}
	return false
}

// ByServiceID implements the Sort interface because Go can't sort without it.
type ByServiceID []*consul.ServiceEntry

func (se ByServiceID) Len() int           { return len(se) }
func (se ByServiceID) Swap(i, j int)      { se[i], se[j] = se[j], se[i] }
func (se ByServiceID) Less(i, j int) bool { return se[i].Service.ID < se[j].Service.ID }
