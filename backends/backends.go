package backends

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// Backend represents a command to execute when another application changes
type Backend struct {
	Name             string      `mapstructure:"name"`
	Poll             int         `mapstructure:"poll"` // time in seconds
	OnChangeExec     interface{} `mapstructure:"onChange"`
	Tag              string      `mapstructure:"tag"`
	Timeout          string      `mapstructure:"timeout"`
	discoveryService discovery.ServiceBackend
	lastState        interface{}
	onChangeCmd      *commands.Command
}

// NewBackends creates a new backend from a raw config structure
func NewBackends(raw []interface{}, disc discovery.ServiceBackend) ([]*Backend, error) {
	if raw == nil {
		return []*Backend{}, nil
	}
	var backends []*Backend
	if err := utils.DecodeRaw(raw, &backends); err != nil {
		return nil, fmt.Errorf("Backend configuration error: %v", err)
	}
	for _, b := range backends {
		if err := utils.ValidateServiceName(b.Name); err != nil {
			return nil, err
		}
		if b.OnChangeExec == nil {
			return nil, fmt.Errorf("`onChange` is required in backend %s",
				b.Name)
		}
		cmd, err := commands.NewCommand(b.OnChangeExec, b.Timeout)
		if err != nil {
			return nil, fmt.Errorf("Could not parse `onChange` in backend %s: %s",
				b.Name, err)
		}
		cmd.Name = fmt.Sprintf("%s.health", b.Name)
		b.onChangeCmd = cmd

		if b.Poll < 1 {
			return nil, fmt.Errorf("`poll` must be > 0 in backend %s",
				b.Name)
		}
		b.onChangeCmd = cmd
		b.discoveryService = disc
	}
	return backends, nil
}

// PollTime implements Pollable for Backend
// It returns the backend's poll interval.
func (b Backend) PollTime() time.Duration {
	return time.Duration(b.Poll) * time.Second
}

// PollAction implements Pollable for Backend.
// If the values in the discovery service have changed since the last run,
// we fire the on change handler.
func (b *Backend) PollAction() {
	if b.CheckForUpstreamChanges() {
		b.OnChange()
	}
}

// PollStop does nothing in a Backend
func (b *Backend) PollStop() {
	// Nothing to do
}

// CheckForUpstreamChanges checks the service discovery endpoint for any changes
// in a dependent backend. Returns true when there has been a change.
func (b *Backend) CheckForUpstreamChanges() bool {
	return b.discoveryService.CheckForUpstreamChanges(b.Name, b.Tag)
}

// OnChange runs the backend's onChange command, returning the results
func (b *Backend) OnChange() error {
	return commands.RunWithTimeout(b.onChangeCmd, log.Fields{
		"process": "onChange", "backend": b.Name})
}
