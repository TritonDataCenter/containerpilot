package subcommands

import (
	"errors"

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
		err := errors.New("Reload: Failed send reload command")
		return err
	}

	return nil
}

func (self Subcommand) SendEnableMaintenance() error {
	_, err := self.client.SetMaintenance(true)
	if err != nil {
		err := errors.New("EnableMaintanence: Failed send client maintanance enable command")
		return err
	}

	return nil
}

func (self Subcommand) SendDisableMaintenance() error {
	_, err := self.client.SetMaintenance(false)
	if err != nil {
		err := errors.New("DisableMaintenance: Failed send client maintanance disable command")
		return err
	}

	return nil
}

func (self Subcommand) SendEnviron(env string) error {
	_, err := self.client.PutEnv(env)
	if err != nil {
		err := errors.New("SendEnviron: Failed send environ command")
		return err
	}

	return nil
}

func (self Subcommand) SendMetric(metrics string) error {
	_, err := self.client.PutMetric(metrics)
	if err != nil {
		err := errors.New("PutMetric: Failed send metric command")
		return err
	}

	return nil
}
