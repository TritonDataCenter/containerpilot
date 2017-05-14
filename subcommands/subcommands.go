package subcommands

import (
	"encoding/json"
	"errors"

	"github.com/joyent/containerpilot/client"
	"github.com/joyent/containerpilot/config"
)

type Subcommand struct {
	client *client.HTTPClient
}

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

func (self Subcommand) SendReload() error {
	_, err := self.client.Reload()
	if err != nil {
		return err
	}

	return nil
}

func (self Subcommand) SendMaintenance(isEnabled bool) error {
	_, err := self.client.SetMaintenance(isEnabled)
	if err != nil {
		return err
	}

	return nil
}

func (self Subcommand) SendEnviron(env map[string]string) error {
	envJSON, err := json.Marshal(env)
	if err != nil {
		return err
	}

	_, err = self.client.PutEnv(string(envJSON))
	if err != nil {
		return err
	}

	return nil
}

func (self Subcommand) SendMetric(metrics map[string]string) error {
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	_, err = self.client.PutMetric(string(metricsJSON))
	if err != nil {
		return err
	}

	return nil
}
