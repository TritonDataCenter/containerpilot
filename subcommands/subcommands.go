package subcommands

import (
	"errors"

	"github.com/joyent/containerpilot/client"
	"github.com/joyent/containerpilot/config"
)

func initClient(configFlag string) error {
	cfg, err := config.LoadConfig(configFlag)
	if err != nil {
		return err
	}

	if cfg.Control == nil {
		err := errors.New("Reload: Couldn't reuse control config")
		return err
	}

	httpclient, err := client.NewHTTPClient(cfg.Control.SocketPath)
	if err != nil {
		return err
	}

	return httpclient
}

func SendReload(configFlag string) error {
	httpclient := initClient(configFlag)
	if err := httpclient.Reload(); err != nil {
		err := errors.New("Reload: Failed send reload command")
		return err
	}

	return nil
}

func SendEnableMaintenance(configFlag string) error {
	httpclient := initClient(configFlag)
	if err := httpclient.SetMaintenance(true); err != nil {
		err := errors.New("EnableMaintanence: Failed send client maintanance enable command")
		return err
	}

	return nil
}

func SendDisableMaintenance(configFlag string) error {
	httpclient := initClient(configFlag)
	if err := httpclient.SetMaintenance(false); err != nil {
		err := errors.New("DisableMaintenance: Failed send client maintanance disable command")
		return err
	}

	return nil
}

func SendEnviron(env string, configFlag string) error {
	httpclient := initClient(configFlag)
	// TODO: Encode environment into JSON map
	if err := httpclient.PutEnv(env); err != nil {
		err := errors.New("SendEnviron: Failed send environ command")
		return err
	}

	return nil
}

func SendMetric(metrics string, configFlag string) error {
	httpclient := initClient(configFlag)
	// TODO: Encode metrics into JSON map
	if err := httpclient.PutMetric(env); err != nil {
		err := errors.New("PutMetric: Failed send metric command")
		return err
	}

	return nil
}
