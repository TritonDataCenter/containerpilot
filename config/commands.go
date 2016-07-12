package config

import (
	"fmt"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
)

func (cfg *rawConfig) parsePreStart() (*exec.Cmd, error) {
	// onStart has been deprecated for preStart. Remove in 2.0
	if cfg.preStart != nil && cfg.onStart != nil {
		log.Warnf("The onStart option has been deprecated in favor of preStart. ContainerPilot will use only the preStart option provided")
	}

	// alias the onStart behavior to preStart
	if cfg.preStart == nil && cfg.onStart != nil {
		log.Warnf("The onStart option has been deprecated in favor of preStart. ContainerPilot will use the onStart option as a preStart")
		cfg.preStart = cfg.onStart
	}
	return parseCommand("preStart", cfg.preStart)
}

func (cfg *rawConfig) parsePreStop() (*exec.Cmd, error) {
	return parseCommand("preStop", cfg.preStop)
}

func (cfg *rawConfig) parsePostStop() (*exec.Cmd, error) {
	return parseCommand("postStop", cfg.postStop)
}

func parseCommand(name string, args interface{}) (*exec.Cmd, error) {
	cmd, err := commands.ParseCommandArgs(args)
	if err != nil {
		return nil, fmt.Errorf("Could not parse `%s`: %s", name, err)
	}
	return cmd, nil
}
