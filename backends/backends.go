package backends

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/discovery"
	"github.com/joyent/containerpilot/utils"
)

// Backend represents a command to execute when another application changes
type Backend struct {
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	OnChangeExec     json.RawMessage `json:"onChange"`
	Tag              string          `json:"tag,omitempty"`
	discoveryService discovery.DiscoveryService
	lastState        interface{}
	onChangeCmd      *exec.Cmd
}

func NewBackends(raw json.RawMessage, disc discovery.DiscoveryService) ([]*Backend, error) {
	if raw == nil {
		return []*Backend{}, nil
	}
	backends := make([]*Backend, 0)
	if err := json.Unmarshal(raw, &backends); err != nil {
		return nil, fmt.Errorf("Backend configuration error: %v", err)
	}
	for _, b := range backends {
		if b.Name == "" {
			return nil, fmt.Errorf("backend must have a `name`")
		}
		cmd, err := utils.ParseCommandArgs(b.OnChangeExec)
		if err != nil {
			return nil, fmt.Errorf("Could not parse `onChange` in backend %s: %s",
				b.Name, err)
		}
		if cmd == nil {
			return nil, fmt.Errorf("`onChange` is required in backend %s",
				b.Name)
		}
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
func (b *Backend) OnChange() (int, error) {
	defer func() {
		// reset command object because it can't be reused
		b.onChangeCmd = utils.ArgsToCmd(b.onChangeCmd.Args)
	}()

	exitCode, err := utils.RunWithFields(b.onChangeCmd, log.Fields{"process": "OnChange", "backend": b.Name})
	return exitCode, err
}
