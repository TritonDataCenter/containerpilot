package subcommands

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/joyent/containerpilot/client"
	"github.com/joyent/containerpilot/config"
)

type Subcommand struct {
	client *client.HTTPClient
}

func Init(configFlag string) (*Subcommand, error) {
	cfg, err := config.LoadConfig(configFlag)
	if err != nil {
		return nil, err
	}

	if cfg.Control == nil {
		err := errors.New("Reload: Couldn't reuse control config")
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

func (self Subcommand) SendReload() error {
	_, err := self.client.Reload()
	if err != nil {
		err := errors.New("Reload: failed send reload command")
		return err
	}

	return nil
}

func (self Subcommand) SendEnableMaintenance() error {
	_, err := self.client.SetMaintenance(true)
	if err != nil {
		err := errors.New("EnableMaintanence: failed send client maintanance enable command")
		return err
	}

	return nil
}

func (self Subcommand) SendDisableMaintenance() error {
	_, err := self.client.SetMaintenance(false)
	if err != nil {
		err := errors.New("DisableMaintenance: failed send client maintanance disable command")
		return err
	}

	return nil
}

func (self Subcommand) SendEnviron(env map[string]string) error {
	envJSON, err := json.Marshal(env)
	if err != nil {
		fmt.Println("SendEnviron: failed to marshal JSON values", err)
	}

	_, err = self.client.PutEnv(string(envJSON))
	if err != nil {
		err := errors.New("SendEnviron: failed send environ command")
		return err
	}

	return nil
}

func (self Subcommand) SendMetric(metrics map[string]string) error {
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		fmt.Println("SendMetric: failed to marshal JSON values", err)
	}

	_, err = self.client.PutMetric(string(metricsJSON))
	if err != nil {
		err := errors.New("SendMetric: failed send metric command")
		return err
	}

	return nil
}
