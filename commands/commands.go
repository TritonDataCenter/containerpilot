package commands

import (
	"context"
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
		// the stdlib almost certainly won't include the underlying error
		// code with this error (if any!). we can try to parse the various
		// totally undocumented strings that come back but I'd rather do my
		// own dentistry. a safe bet is that the end user has given us an
		// invalid executable so we're going to return 127.
		log.Errorln(err)
		return 127, err
	}
	os.Setenv(
		fmt.Sprintf("CONTAINERPILOT_%s_PID", strings.ToUpper(c.Name)),
		fmt.Sprintf("%v", c.Cmd.Process.Pid),
	)
	defer log.Debugf("%s.RunAndWait end", c.Name)
	if code, err := c.wait(); err != nil {
		log.Errorf("%s exited with error: %v", c.Name, err)
		return code, err
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), c.TimeoutDuration)
	defer cancel()

	log.Debugf("%s.RunWithTimeout start", c.Name)
	c.setUpCmd(fields)
	defer c.closeLogs()
	log.Debugf("%s.Cmd start", c.Name)
	if err := c.Cmd.Start(); err != nil {
		log.Errorf("unable to start %s: %v", c.Name, err)
		return err
	}
	go func() {
		select {
		case <-ctx.Done():
			log.Warnf("%s timeout after %s: '%s'", c.Name, c.Timeout, c.Args)
			c.Kill()
		}
	}()

	defer log.Debugf("%s.RunWithTimeout end", c.Name)
	if _, err := c.wait(); err != nil {
		log.Errorf("%s exited with error: %v", c.Name, err)
		return err
	}
	return nil
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
		if err.Error() == errNoChild {
			log.Debugf(err.Error())
			return 0, nil // process exited cleanly before we hit wait4
		}
		return 1, err
	}
	return 0, nil
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

	// assign a unique process group ID so we can kill all
	// its children on timeout
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cmd = cmd
}

// Kill sends a kill signal to the underlying process.
func (c *Command) Kill() {
	log.Debugf("%s.kill", c.Name)
	if c.Cmd != nil && c.Cmd.Process != nil {
		log.Warnf("killing command at pid: %d", c.Cmd.Process.Pid)
		syscall.Kill(-c.Cmd.Process.Pid, syscall.SIGKILL)
	}
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
