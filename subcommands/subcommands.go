package subcommands

import (
	"errors"

	"github.com/joyent/containerpilot/client"
	"github.com/joyent/containerpilot/config"
)

func SendReload(configFlag string) error {
	cfg, err := config.LoadConfig(configFlag)
	if err != nil {
		return err
	}

	if cfg.Control == nil {
		err := errors.New("Reload: Couldn't reuse control config")
		return err
	}

	c, err := client.NewHTTPClient(cfg.Control.SocketPath)
	if err != nil {
		return err
	}

	if err := c.Reload(); err != nil {
		err := errors.New("Reload: Failed to call client reload command")
		return err
	}

	return nil
}

// func SendMaintenance() {}
// func SendEnviron() {}
// func SendMetric() {}
