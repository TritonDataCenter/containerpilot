// Package commands provides a wrapper around os/exec to consistently
// manage process execution, cancellation of their child processes,
// timeouts, logging, arg parsing, and correct shutdown.
package commands

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/joyent/containerpilot/events"
	log "github.com/sirupsen/logrus"
)

// Command wraps an os/exec.Cmd with a timeout, logging, and arg parsing.
type Command struct {
	Name    string // this gets used only in logs, defaults to Exec
	Cmd     *exec.Cmd
	Exec    string
	Args    []string
	Timeout time.Duration
	logger  log.Entry
	lock    *sync.Mutex
}

// NewCommand parses JSON config into a Command
func NewCommand(rawArgs interface{}, timeout time.Duration, fields log.Fields) (*Command, error) {
	exec, args, err := ParseArgs(rawArgs)
	if err != nil {
		return nil, err
	}
	logger := log.WithFields(fields)
	cmd := &Command{
		Name:    exec, // override this in caller
		Exec:    exec,
		Args:    args,
		Timeout: timeout,
		lock:    &sync.Mutex{},
		logger:  *logger,
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

	cmd := ArgsToCmd(c.Exec, c.Args)
	cmd.Stdout = c.logger.Writer()
	cmd.Stderr = c.logger.Writer()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cmd = cmd
	ctx, cancel := getContext(pctx, c.Timeout)

	go func() {
		// Children may have side-effects so we don't want to wait for them
		// to be reaped if we timeout, so we can't use CommandContext from
		// the stdlib. Instead we've assigned a unique process group ID, so
		// here we'll block until the context is done and then unlock and kill
		// all child processes.
		<-ctx.Done()
		defer c.lock.Unlock()
		if ctx.Err() == context.DeadlineExceeded {
			log.Warnf("%s timeout after %s: '%s'", c.Name, c.Timeout, c.Args)
			c.Kill()
			return
		}
		c.Term()
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
		// blocks this goroutine here; if the context gets cancelled
		// we'll return from Wait() and publish events
		if err := c.Cmd.Wait(); err != nil {
			log.Errorf("%s exited with error: %v", c.Name, err)
			bus.Publish(events.Event{events.ExitFailed, c.Name})
			bus.Publish(events.Event{events.Error,
				fmt.Errorf("%s: %s", c.Name, err).Error()})
		} else {
			log.Debugf("%s exited without error", c.Name)
			bus.Publish(events.Event{events.ExitSuccess, c.Name})
		}
	}()
}

func getContext(pctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(pctx, timeout)
	}
	return context.WithCancel(pctx)
}

// Kill sends a kill signal to the underlying process if it still exists,
// as well as all its children
func (c *Command) Kill() {
	log.Debugf("%s.kill", c.Name)
	if c.Cmd != nil && c.Cmd.Process != nil {
		log.Debugf("killing command '%v' at pid: %d", c.Name, c.Cmd.Process.Pid)
		syscall.Kill(-c.Cmd.Process.Pid, syscall.SIGKILL)
	}
}

// Term sends a terminate signal to the underlying process if it still exists,
// as well as all its children
func (c *Command) Term() {
	log.Debugf("%s.term", c.Name)
	if c.Cmd != nil && c.Cmd.Process != nil {
		log.Debugf("terminating command '%v' at pid: %d", c.Name, c.Cmd.Process.Pid)
		syscall.Kill(-c.Cmd.Process.Pid, syscall.SIGTERM)
	}
}
