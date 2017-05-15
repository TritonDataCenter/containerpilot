package subcommands

import (
	"encoding/json"
	"errors"

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
	var socketPath = "/var/run/containerpilot.socket"

	if configFlag != "" {
		cfg, err := config.LoadConfig(configFlag)
		if err != nil {
			return nil, err
		}
		if cfg.Control == nil {
			err := errors.New("Reload: Couldn't reuse control config")
			return nil, err
		}
		socketPath = cfg.Control.SocketPath
	}

	httpclient, err := client.NewHTTPClient(socketPath)
	if err != nil {
		return nil, err
	}

	return &Subcommand{
		httpclient,
	}, nil
}

// SendReload fires a Reload request through the HTTPClient.
func (s Subcommand) SendReload() error {
	_, err := s.client.Reload()
	if err != nil {
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

	_, err := s.client.SetMaintenance(flag)
	if err != nil {
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

	_, err = s.client.PutEnv(string(envJSON))
	if err != nil {
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

	_, err = s.client.PutMetric(string(metricsJSON))
	if err != nil {
		return err
	}

	return nil
}
