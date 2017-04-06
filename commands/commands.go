package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/events"
)

const (
	errWaitNoChild   = "wait: no child processes"
	errWaitIDNoChild = "waitid: no child processes"
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
	} // exec.Cmd created at Run or RunAndWaitForOutput
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
			// canceled it in the caller or the applicaton exited gracefully,
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
		// blocks this goroutine here; if the context gets cancelled
		// we'll return from wait() and do all the cleanup
		if _, err := c.wait(); err != nil {
			log.Errorf("%s exited with error: %v", c.Name, err)
			bus.Publish(events.Event{events.ExitFailed, c.Name})
			bus.Publish(events.Event{events.Error, err.Error()})
		} else {
			log.Debugf("%s exited without error", c.Name)
			bus.Publish(events.Event{events.ExitSuccess, c.Name})
		}
	}()
}

// RunAndWaitForOutput runs the command and blocks until completed, then
// returns a string of the stdout
// TODO v3: remove this once the control plane is available for Sensors (the
// only caller) to send metrics to
func (c *Command) RunAndWaitForOutput(pctx context.Context, bus *events.EventBus) string {
	if c == nil {
		log.Debugf("nothing to run for %s", c.Name)
		return ""
	}
	// we should never have more than one instance running for any
	// realistic configuration but this ensures that's the case
	c.lock.Lock()
	log.Debugf("%s.Run start", c.Name)
	c.setUpCmd()
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if c.Timeout > 0 {
		ctx, cancel = context.WithTimeout(pctx, c.Timeout)
	} else {
		ctx, cancel = context.WithCancel(pctx)
	}
	defer cancel()

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
			// canceled it in the caller or the applicaton exited gracefully,
			// so Kill() will have to handle both cases safely
			c.Kill()
		}
	}()
	// we'll pass stderr to the container's stderr, but stdout must
	// be "clean" and not have anything other than what we intend
	// to write to our collector.
	c.Cmd.Stderr = os.Stderr
	log.Debugf("%s.Cmd.Output", c.Name)
	defer log.Debugf("%s.RunAndWaitForOutput end", c.Name)
	// blocks this goroutine here; if the context gets cancelled
	// we'll return from wait() and do all the cleanup
	out, err := c.Cmd.Output()
	if err != nil {
		log.Errorf("%s exited with error: %v", c.Name, err)
		bus.Publish(events.Event{events.ExitFailed, c.Name})
		bus.Publish(events.Event{events.Error, err.Error()})
		return ""
	}
	bus.Publish(events.Event{events.ExitSuccess, c.Name})
	return string(out[:])
}

func (c *Command) wait() (int, error) {
	waitStatus, err := c.Cmd.Process.Wait()
	if waitStatus != nil && !waitStatus.Success() {
		var returnStatus = 1
		if status, ok := waitStatus.Sys().(syscall.WaitStatus); ok {
			returnStatus = status.ExitStatus()
		}
		return returnStatus, fmt.Errorf("%s: %s", c.Name, waitStatus)
	} else if err != nil {
		if err.Error() == errWaitNoChild || err.Error() == errWaitIDNoChild {
			log.Debugf(err.Error())
			return 0, nil // process exited cleanly before we hit wait4
		}
		return 1, err
	}
	return 0, nil
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
	if c != nil && c.logger != nil {
		c.logger.Close()
	}
	return
}
