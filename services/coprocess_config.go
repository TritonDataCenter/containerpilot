package services

/*
TODO: this entire file will be eliminated after we update config syntax
*/

import (
	"fmt"
	"strings"

	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/utils"
)

// CoprocessConfig configures a non-advertised service.
type CoprocessConfig struct {
	Name     string      `mapstructure:"name"`
	Exec     interface{} `mapstructure:"command"`
	Restarts interface{} `mapstructure:"restarts"`
}

// NewCoprocessConfigs parses json config into a validated slice of ServiceConfigs.
func NewCoprocessConfigs(raw []interface{}) ([]*ServiceConfig, error) {
	var (
		coprocesses []*CoprocessConfig
		services    []*ServiceConfig
	)
	if raw == nil {
		return services, nil
	}
	if err := utils.DecodeRaw(raw, &coprocesses); err != nil {
		return nil, fmt.Errorf("coprocess configuration error: %v", err)
	}
	for _, cfg := range coprocesses {
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		service := cfg.ToServiceConfig()
		if err := service.Validate(nil); err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}

// ToServiceConfig ...
func (co *CoprocessConfig) ToServiceConfig() *ServiceConfig {
	service := &ServiceConfig{
		Name:     co.Name,
		Exec:     co.Exec,
		Restarts: co.Restarts,
	}
	return service
}

// Validate ...
func (co *CoprocessConfig) Validate() error {
	if co.Exec == nil {
		return fmt.Errorf("coprocess did not provide a command")
	}
	if co.Name == "" {
		exec, cmdArgs, err := commands.ParseArgs(co.Exec)
		if err != nil {
			return err
		}
		args := append([]string{exec}, cmdArgs...)
		co.Name = strings.Join(args, " ")
	}
	return nil
}
