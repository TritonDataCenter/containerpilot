package config

import (
	"fmt"

	"github.com/joyent/containerpilot/commands"
)

func (cfg *rawConfig) parsePreStart() (*commands.Command, error) {
	return parseCommand("preStart", cfg.preStart)
}

func (cfg *rawConfig) parsePreStop() (*commands.Command, error) {
	return parseCommand("preStop", cfg.preStop)
}

func (cfg *rawConfig) parsePostStop() (*commands.Command, error) {
	return parseCommand("postStop", cfg.postStop)
}

func parseCommand(name string, args interface{}) (*commands.Command, error) {
	if args == nil {
		return nil, nil
	}
	cmd, err := commands.NewCommand(args, "0")
	if err != nil {
		return nil, fmt.Errorf("Could not parse `%s`: %s", name, err)
	}
	cmd.Name = name
	return cmd, nil
}
