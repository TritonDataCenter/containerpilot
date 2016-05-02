package config

import (
	"fmt"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

// ParsePreStart ...
func (cfg *Config) ParsePreStart() (*exec.Cmd, error) {
	// onStart has been deprecated for preStart. Remove in 2.0
	if cfg.PreStart != nil && cfg.OnStart != nil {
		log.Warnf("The onStart option has been deprecated in favor of preStart. ContainerPilot will use only the preStart option provided")
	}

	// alias the onStart behavior to preStart
	if cfg.PreStart == nil && cfg.OnStart != nil {
		log.Warnf("The onStart option has been deprecated in favor of preStart. ContainerPilot will use the onStart option as a preStart")
		cfg.PreStart = cfg.OnStart
	}
	return parseCommand("preStart", cfg.PreStart)
}

// ParsePreStop ...
func (cfg *Config) ParsePreStop() (*exec.Cmd, error) {
	return parseCommand("preStop", cfg.PreStop)
}

// ParsePostStop ...
func (cfg *Config) ParsePostStop() (*exec.Cmd, error) {
	return parseCommand("postStop", cfg.PostStop)
}

func parseCommand(name string, args interface{}) (*exec.Cmd, error) {
	cmd, err := utils.ParseCommandArgs(args)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `%s`: %s", name, err)
	}
	return cmd, nil
}
