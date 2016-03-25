package containerbuddy

import (
	"encoding/json"
	"os/exec"
)

// BackendConfig represents a command to execute when another application changes
type BackendConfig struct {
	Name             string          `json:"name"`
	Poll             int             `json:"poll"` // time in seconds
	OnChangeExec     json.RawMessage `json:"onChange"`
	Tag              string          `json:"tag,omitempty"`
	discoveryService DiscoveryService
	lastState        interface{}
	onChangeCmd      *exec.Cmd
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
	return b.discoveryService.CheckForUpstreamChanges(b)
}

// OnChange runs the backend's onChange command, returning the results
func (b *BackendConfig) OnChange() (int, error) {
	exitCode, err := run(b.onChangeCmd)
	// Reset command object - since it can't be reused
	b.onChangeCmd = argsToCmd(b.onChangeCmd.Args)
	return exitCode, err
}
