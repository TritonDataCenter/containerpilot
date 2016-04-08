package backends

import (
	"discovery"
	"encoding/json"
	"fmt"
	"os/exec"
	"utils"
)

// BackendConfig represents a command to execute when another application changes
type BackendConfig struct {
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	OnChangeExec     json.RawMessage `json:"onChange"`
	Tag              string          `json:"tag,omitempty"`
	discoveryService discovery.DiscoveryService
	lastState        interface{}
	onChangeCmd      *exec.Cmd
}

func NewBackends(raw json.RawMessage, disc discovery.DiscoveryService) ([]*BackendConfig, error) {
	if raw == nil {
		return []*BackendConfig{}, nil
	}
	backends := make([]*BackendConfig, 0)
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

// PollTime implements Pollable for BackendConfig
// It returns the backend's poll interval.
func (b BackendConfig) PollTime() int {
	return b.Poll
}

// PollAction implements Pollable for BackendConfig.
// If the values in the discovery service have changed since the last run,
// we fire the on change handler.
func (b BackendConfig) PollAction() {
	if b.CheckForUpstreamChanges() {
		b.OnChange()
	}
}

// CheckForUpstreamChanges checks the service discovery endpoint for any changes
// in a dependent backend. Returns true when there has been a change.
func (b *BackendConfig) CheckForUpstreamChanges() bool {
	return b.discoveryService.CheckForUpstreamChanges(b.Name, b.Tag)
}

// OnChange runs the backend's onChange command, returning the results
func (b *BackendConfig) OnChange() (int, error) {
	defer func() {
		// reset command object because it can't be reused
		b.onChangeCmd = utils.ArgsToCmd(b.onChangeCmd.Args)
	}()
	exitCode, err := utils.Run(b.onChangeCmd)
	return exitCode, err
}
