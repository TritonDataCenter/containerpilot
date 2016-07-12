package commands

import (
	"os/exec"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

// RunWithFields executes the given command and blocks until completed
// Returns the exit code and error message (if any).
// Logs output streams to the log driver instead of stdout/stderr
// - stderr creates DEBUG level entries
// - stdout creates INFO level entries
// Adds the given fields to the log entries
func RunWithFields(cmd *exec.Cmd, fields log.Fields) (int, error) {
	if cmd != nil {
		stderr := utils.NewLogWriter(fields, log.DebugLevel)
		stdout := utils.NewLogWriter(fields, log.InfoLevel)
		cmd.Stderr = stderr
		cmd.Stdout = stdout
		defer stderr.Close()
		defer stdout.Close()
	}
	return ExecuteAndWait(cmd)
}

// Run executes the given command and blocks until completed
// Returns the exit code and error message (if any).
// Logs output streams to the log driver instead of stdout/stderr
// - stderr creates DEBUG level entries
// - stdout creates INFO level entries
func Run(cmd *exec.Cmd) (int, error) {
	return RunWithFields(cmd, nil)
}

// ExecuteAndWait runs the given command and blocks until completed
func ExecuteAndWait(cmd *exec.Cmd) (int, error) {
	if cmd == nil {
		return 0, nil
	}
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
