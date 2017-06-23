package commands

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
)

// Command wraps an os/exec.Cmd with a timeout, logging, and arg parsing.
type Command struct {
	Name      string // this gets used only in logs, defaults to Exec
	Cmd       *exec.Cmd
	Exec      string
	Args      []string
	Timeout   time.Duration
	logger    io.WriteCloser
	logFields log.Fields
	lock      *sync.Mutex
}

// NewCommand parses JSON config into a Command
func NewCommand(rawArgs interface{}, timeout time.Duration, fields log.Fields) (*Command, error) {
	exec, args, err := ParseArgs(rawArgs)
	if err != nil {
		return nil, err
	}
	cmd := &Command{
		Name:      exec, // override this in caller
		Exec:      exec,
		Args:      args,
		Timeout:   timeout,
		lock:      &sync.Mutex{},
		logger:    log.StandardLogger().Writer(),
		logFields: fields,
	} // exec.Cmd created at Run
	return cmd, nil
}

// Run creates an exec.Cmd for the Command and runs it asynchronously.
// If the parent context is closed/canceled this will terminate the
// child process and do any cleanup we need.
func (c *Command) Run(pctx context.Context, bus *events.EventBus) {
	if c == nil {
		log.Debugf("nothing to run for %s", c.Name)
		return
	}
	// we should never have more than one instance running for any
	// realistic configuration but this ensures that's the case
	c.lock.Lock()
	log.Debugf("%s.Run start", c.Name)
	c.setUpCmd()
	c.Cmd.Stdout = c.logger
	c.Cmd.Stderr = c.logger

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if c.Timeout > 0 {
		ctx, cancel = context.WithTimeout(pctx, c.Timeout)
	} else {
		ctx, cancel = context.WithCancel(pctx)
	}

	go func() {
		select {
		case <-ctx.Done():
			// unlock only here because we'll receive this
			// cancel from both a typical exit and a timeout
			defer c.lock.Unlock()
			if ctx.Err() == context.DeadlineExceeded {
				log.Warnf("%s timeout after %s: '%s'", c.Name, c.Timeout, c.Args)
			}
			// if the context was canceled we don't know if its because we
			// canceled it in the caller or the application exited gracefully,
			// so Kill() will have to handle both cases safely
			c.Kill()
		}
	}()

	go func() {
		defer cancel()
		defer log.Debugf("%s.Run end", c.Name)
		if err := c.Cmd.Start(); err != nil {
			log.Errorf("unable to start %s: %v", c.Name, err)
			bus.Publish(events.Event{events.ExitFailed, c.Name})
			bus.Publish(events.Event{events.Error, err.Error()})
			return
		}
		// See https://golang.org/src/syscall/exec_linux.go
		// SysProcAttr.Setpgid:
		// "Set process group ID to Pgid, or, if Pgid == 0, to new pid"
		// So in the case where ContainerPilot is PID1 this will fail
		// to reap zombies unless we pass the PID and not the PGID to
		// the syscall.Wait4 in reapChildren
		pgid := c.Cmd.SysProcAttr.Pgid
		if pgid == 0 {
			pgid = c.Cmd.Process.Pid
		}
		defer reapChildren(pgid)

		// blocks this goroutine here; if the context gets cancelled
		// we'll return from wait() and do all the cleanup
		if err := c.wait(); err != nil {
			log.Errorf("%s exited with error: %v", c.Name, err)
			bus.Publish(events.Event{events.ExitFailed, c.Name})
			bus.Publish(events.Event{events.Error, err.Error()})
		} else {
			log.Debugf("%s exited without error", c.Name)
			bus.Publish(events.Event{events.ExitSuccess, c.Name})
		}
	}()
}

func (c *Command) wait() error {
	err := c.Cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.ExitStatus() == 0 {
					return nil
				}
			}
		}
		return fmt.Errorf("%s: %s", c.Name, err.Error())
	}
	return nil
}

func (c *Command) setUpCmd() {
	cmd := ArgsToCmd(c.Exec, c.Args)

	// assign a unique process group ID so we can kill all
	// its children on timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cmd = cmd
}

// Kill sends a kill signal to the underlying process, if it still exists
func (c *Command) Kill() {
	log.Debugf("%s.kill", c.Name)
	if c.Cmd != nil && c.Cmd.Process != nil {
		log.Debugf("killing command '%v' at pid: %d", c.Name, c.Cmd.Process.Pid)
		syscall.Kill(-c.Cmd.Process.Pid, syscall.SIGKILL)
	}
}

// CloseLogs safely closes the io.WriteCloser we're using to pipe logs
func (c *Command) CloseLogs() {
	// need to nil check these because they might have been closed
	// concurrently.
	if c != nil && c.logger != nil {
		c.logger.Close()
	}
	return
}
