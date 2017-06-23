package subcommands

import (
	"encoding/json"

	"github.com/joyent/containerpilot/client"
	"github.com/joyent/containerpilot/config"
)

// Subcommand provides a simple object for storing a configured HTTPClient.
type Subcommand struct {
	client *client.HTTPClient
}

// Init initializes the configuration of a Subcommand function and the
// HTTPClient which they utilize for control plane interaction.
func Init(configFlag string) (*Subcommand, error) {
	cfg, err := config.LoadConfig(configFlag)
	if err != nil {
		return nil, err
	}

	httpclient, err := client.NewHTTPClient(cfg.Control.SocketPath)
	if err != nil {
		return nil, err
	}

	return &Subcommand{
		httpclient,
	}, nil
}

// SendReload fires a Reload request through the HTTPClient.
func (s Subcommand) SendReload() error {
	if err := s.client.Reload(); err != nil {
		return err
	}

	return nil
}

// SendMaintenance fires either an enable or disable SetMaintenance request
// through the HTTPClient.
func (s Subcommand) SendMaintenance(isEnabled string) error {
	flag := false
	if isEnabled == "enable" {
		flag = true
	}

	if err := s.client.SetMaintenance(flag); err != nil {
		return err
	}

	return nil
}

// SendEnviron fires a PutEnv request through the HTTPClient.
func (s Subcommand) SendEnviron(env map[string]string) error {
	envJSON, err := json.Marshal(env)
	if err != nil {
		return err
	}

	if err = s.client.PutEnv(string(envJSON)); err != nil {
		return err
	}

	return nil
}

// SendMetric fires a PutMetric request through the HTTPClient.
func (s Subcommand) SendMetric(metrics map[string]string) error {
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	if err = s.client.PutMetric(string(metricsJSON)); err != nil {
		return err
	}

	return nil
}

// GetPing fires a ping check through the HTTPClient.
func (s Subcommand) GetPing() error {
	return s.client.GetPing()
}
