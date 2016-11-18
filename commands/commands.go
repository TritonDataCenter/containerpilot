package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

const errNoChild = "wait: no child processes"

// Command wraps an os/exec.Cmd with a timeout, logging, and arg parsing.
type Command struct {
	Name            string // this gets used only in logs, defaults to Exec
	Cmd             *exec.Cmd
	Exec            string
	Args            []string
	Timeout         string
	TimeoutDuration time.Duration
	ticker          *time.Ticker
	logWriters      []io.WriteCloser
}

// NewCommand parses JSON config into a Command
func NewCommand(rawArgs interface{}, timeoutFmt string) (*Command, error) {
	exec, args, err := ParseArgs(rawArgs)
	if err != nil {
		return nil, err
	}
	timeout, err := getTimeout(timeoutFmt)
	if err != nil {
		return nil, err
	}
	cmd := &Command{
		Name:            exec, // override this in caller
		Exec:            exec,
		Args:            args,
		Timeout:         timeoutFmt,
		TimeoutDuration: timeout,
	} // cmd, ticker, logWriters all created at RunAndWait or RunWithTimeout
	return cmd, nil
}

func getTimeout(timeoutFmt string) (time.Duration, error) {
	if timeoutFmt != "" {
		timeout, err := utils.ParseDuration(timeoutFmt)
		if err != nil {
			return time.Duration(0), err
		}
		return timeout, nil
	}
	// support commands that don't have a timeout for backwards
	// compatibility
	return time.Duration(0), nil
}

// RunAndWait runs the given command and blocks until completed
func RunAndWait(c *Command, fields log.Fields) (int, error) {
	if c == nil {
		// sometimes this will be ok but we should return an error
		// anyway in case the caller cares
		return 1, errors.New("Command for RunAndWait was nil")
	}
	log.Debugf("%s.RunAndWait start", c.Name)
	c.setUpCmd(fields)
	if fields == nil {
		c.Cmd.Stdout = os.Stdout
		c.Cmd.Stderr = os.Stderr
	}
	log.Debugf("%s.Cmd.Run", c.Name)
	if err := c.Cmd.Start(); err != nil {
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
	os.Setenv(
		fmt.Sprintf("CONTAINERPILOT_%s_PID", strings.ToUpper(c.Name)),
		fmt.Sprintf("%v", c.Cmd.Process.Pid),
	)
	state, err := c.Cmd.Process.Wait()
	if err != nil || (state != nil && !state.Success()) {
		if status, ok := state.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), err
		}
	}
	log.Debugf("%s.RunAndWait end", c.Name)
	return 0, nil
}

// RunAndWaitForOutput runs the given command and blocks until
// completed, then returns the stdout
func RunAndWaitForOutput(c *Command) (string, error) {
	if c == nil {
		// sometimes this will be ok but we should return an error
		// anyway in case the caller cares
		return "", errors.New("Command for RunAndWaitForOutput was nil")
	}
	log.Debugf("%s.RunAndWaitForOutput start", c.Name)
	c.setUpCmd(nil)

	// we'll pass stderr to the container's stderr, but stdout must
	// be "clean" and not have anything other than what we intend
	// to write to our collector.
	c.Cmd.Stderr = os.Stderr
	log.Debugf("%s.Cmd.Output", c.Name)
	out, err := c.Cmd.Output()
	if err != nil {
		return "", err
	}
	log.Debugf("%s.RunAndWaitForOutput end", c.Name)
	return string(out[:]), nil
}

// RunWithTimeout runs the given command and blocks until completed
// or until the timeout expires
func RunWithTimeout(c *Command, fields log.Fields) error {
	if c == nil {
		// sometimes this will be ok but we should return an error
		// anyway in case the caller cares
		return errors.New("Command for RunWithTimeout was nil")
	}
	log.Debugf("%s.RunWithTimeout start", c.Name)
	c.setUpCmd(fields)
	defer c.closeLogs()
	log.Debugf("%s.Cmd.Start", c.Name)
	if err := c.Cmd.Start(); err != nil {
		log.Errorf("Unable to start %s: %v", c.Name, err)
		return err
	}

	err := c.waitForTimeout()
	log.Debugf("%s.RunWithTimeout end", c.Name)
	return err
}

func (c *Command) setUpCmd(fields log.Fields) {
	cmd := ArgsToCmd(c.Exec, c.Args)
	if fields != nil {
		stdout := utils.NewLogWriter(fields, log.InfoLevel)
		stderr := utils.NewLogWriter(fields, log.DebugLevel)
		c.logWriters = []io.WriteCloser{stdout, stderr}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
	}
	c.Cmd = cmd
}

// Kill sends a kill signal to the underlying process.
func (c *Command) Kill() error {
	log.Debugf("%s.kill", c.Name)
	if c.Cmd != nil && c.Cmd.Process != nil {
		log.Warnf("killing command at pid: %d", c.Cmd.Process.Pid)
		return c.Cmd.Process.Kill()
	}
	return nil
}

func (c *Command) waitForTimeout() error {

	quit := make(chan int)
	cmd := c.Cmd

	// for commands that don't have a timeout we just block forever;
	// this is required for backwards compat.
	doTimeout := c.TimeoutDuration != time.Duration(0)
	if doTimeout {
		// wrap a timer in a goroutine and kill the child process
		// if the timer expires
		ticker := time.NewTicker(c.TimeoutDuration)
		go func() {
			defer ticker.Stop()
			select {
			case <-ticker.C:
				log.Warnf("%s timeout after %s: '%s'", c.Name, c.Timeout, c.Args)
				if err := c.Kill(); err != nil {
					log.Errorf("error killing command: %v", err)
					return
				}
				log.Debugf("%s.run#gofunc swallow quit", c.Name)
				// Swallow quit signal
				<-quit
				log.Debugf("%s.run#gofunc swallow quit complete", c.Name)
				return
			case <-quit:
				log.Debugf("%s.run#gofunc received quit", c.Name)
				return
			}
		}()
		// if we send on this when we haven't set up the receiver
		// we'll deadlock
		defer func() { quit <- 0 }()
	}
	log.Debugf("%s.run waiting for PID %d: ", c.Name, cmd.Process.Pid)
	state, err := cmd.Process.Wait()
	if err != nil {
		if err.Error() == errNoChild {
			log.Debugf(err.Error())
			return nil // process exited cleanly before we hit wait4
		}
		log.Errorf("%s exited with error: %v", c.Name, err)
		return err
	}
	if state != nil && !state.Success() {
		return fmt.Errorf("%s exited with error", c.Name)
	}

	log.Debugf("%s.run complete", c.Name)
	return nil
}

func (c *Command) closeLogs() {
	if c.logWriters == nil {
		return
	}
	for _, w := range c.logWriters {
		if err := w.Close(); err != nil {
			log.Errorf("unable to close log writer : %v", err)
		}
	}
}
