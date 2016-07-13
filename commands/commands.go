package commands

import (
	"io"
	"os/exec"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

// Command wraps an os/exec.Cmd with a timeout, logging, and arg parsing.
type Command struct {
	Name            string // this gets used only in logs, defaults to Exec
	Cmd             *exec.Cmd
	Exec            string
	Args            []string
	Timeout         string
	timeoutDuration time.Duration
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
		timeoutDuration: timeout,
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
func (c *Command) RunAndWait() (int, error) {
	cmd := ArgsToCommand(c.Exec, c.Args)
	if cmd == nil {
		return 0, nil
	}
	c.Cmd = cmd
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

// RunWithTimeout runs the given command and blocks until completed
// or until the timeout expires
func (c *Command) RunWithTimeout(fields log.Fields) error {
	log.Debugf("%s.RunWithTimeout start", c.Name)
	cmd := ArgsToCommand(c.Exec, c.Args)
	c.Cmd = cmd
	if fields == nil {
		fields = log.Fields{}
	}
	stdout := utils.NewLogWriter(fields, log.InfoLevel)
	stderr := utils.NewLogWriter(fields, log.DebugLevel)
	c.logWriters = []io.WriteCloser{stdout, stderr}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	defer c.closeLogs()
	log.Debugf("%s.Cmd.Start", c.Name)
	if err := cmd.Start(); err != nil {
		log.Errorf("Unable to start %s: %v", c.Name, err)
		return err
	}

	err := c.waitForTimeout()
	log.Debugf("%s.RunWithTimeout start", c.Name)
	return err
}

func (c *Command) kill() error {
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
	doTimeout := c.timeoutDuration != time.Duration(0)
	if doTimeout {
		// wrap a timer in a goroutine and kill the child process
		// if the timer expires
		ticker := time.NewTicker(c.timeoutDuration)
		go func() {
			defer ticker.Stop()
			select {
			case <-ticker.C:
				log.Warnf("%s timeout after %s: '%s'", c.Name, c.Timeout, c.Args)
				if err := c.kill(); err != nil {
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
	}
	log.Debugf("%s.run waiting for PID %d: ", c.Name, cmd.Process.Pid)
	if _, err := cmd.Process.Wait(); err != nil {
		log.Errorf("%s exited with error: %v", c.Name, err)
		return err
	}
	if doTimeout {
		// if we send on this when we haven't set up the receiver
		// we'll deadlock
		log.Debugf("%s.run sent timeout quit", c.Name)
		quit <- 0
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
