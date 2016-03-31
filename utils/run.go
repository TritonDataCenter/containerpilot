package utils

import (
	"os"
	"os/exec"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

// Executes the given command and blocks until completed
// Returns the exit code and error message (if any).
// Logs errors
func Run(cmd *exec.Cmd) (int, error) {
	code, err := ExecuteAndWait(cmd)
	if err != nil {
		log.Errorln(err)
	}
	return code, err
}

// Executes the given command and blocks until completed
func ExecuteAndWait(cmd *exec.Cmd) (int, error) {
	if cmd == nil {
		return 0, nil
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), err
			}
		}
		// Should only happen if we misconfigure or there's some more
		// serious problem with the underlying open/exec syscalls. But
		// we'll let the lack of heartbeat tell us if something has gone
		// wrong to that extent.
		log.Errorln(err)
		return 1, err
	}
	return 0, nil
}
