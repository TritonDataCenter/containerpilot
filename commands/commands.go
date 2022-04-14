// Package commands provides a wrapper around os/exec to consistently
// manage process execution, cancellation of their child processes,
// timeouts, logging, arg parsing, and correct shutdown.
package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
	fields  log.Fields
	UID     int
	GID     int
}

// NewCommand parses JSON config into a Command
func NewCommand(rawArgs interface{}, timeout time.Duration, fields log.Fields) (*Command, error) {
	exec, args, err := ParseArgs(rawArgs)
	if err != nil {
		return nil, err
	}
	cmd := &Command{
		Name:    exec, // override this in caller
		Exec:    exec,
		Args:    args,
		Timeout: timeout,
		lock:    &sync.Mutex{},
	} // exec.Cmd created at Run

	if fields != nil {
		cmd.fields = fields
		// don't attach the logger if we don't have fields set, so that
		// we can pass-thru the logs raw
		cmd.logger = *log.WithFields(cmd.fields)
	}
	return cmd, nil
}

// EnvName formats Name for use as an environment variable name (PID).
func (c *Command) EnvName() string {
	if c.Name == "" {
		return c.Name
	}

	var name string
	name = filepath.Base(c.Name)

	// remove command extension if exec was used as name
	if strings.Contains(name, ".") {
		name = strings.Replace(name, filepath.Ext(name), "", 1)
	}

	// convert all non-alphanums into an underscore
	matchSyms := regexp.MustCompile("[^[:alnum:]]+")
	name = matchSyms.ReplaceAllString(name, "_")

	// compact multiple underscores into singles
	matchScores := regexp.MustCompile("__+")
	name = matchScores.ReplaceAllString(name, "_")

	return strings.ToUpper(name)
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

	cmd := exec.Command(c.Exec, c.Args...)
	if c.logger.Logger != nil {
		cmd.Stdout = c.logger.Writer()
		cmd.Stderr = c.logger.Writer()
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if os.Getuid() == 0 {
	    if c.UID != 0 && c.GID != 0 {
            cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(c.UID), Gid: uint32(c.GID)}
        } else if c.UID != 0 {
            cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(c.UID)}
        } else if c.GID != 0 {
            cmd.SysProcAttr.Credential = &syscall.Credential{Gid: uint32(c.GID)}
        }
   	} else {
   	    log.Debugf("%s.Skipping uid and gid (ContainerPilot is not running as root)", c.Name)
   	}

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
			bus.Publish(events.Event{Code: events.ExitFailed, Source: c.Name})
			bus.Publish(events.Event{Code: events.Error, Source: err.Error()})
			return
		}

		// if we're able to, log the PID of our Command's exec process through
		// our logger fields
		if c.Cmd != nil && c.Cmd.Process != nil {
			pid := c.Cmd.Process.Pid

			envName := fmt.Sprintf("CONTAINERPILOT_%s_PID", c.EnvName())
			os.Setenv(envName, strconv.Itoa(pid))
			defer os.Unsetenv(envName)

			if len(c.fields) > 0 {
				c.fields["pid"] = pid
				c.logger = *log.WithFields(c.fields)
			}
		}

		// blocks this goroutine here; if the context gets cancelled
		// we'll return from Wait() and publish events
		if err := c.Cmd.Wait(); err != nil {
			log.Errorf("%s exited with error: %v", c.Name, err)
			bus.Publish(events.Event{Code: events.ExitFailed, Source: c.Name})
			bus.Publish(events.Event{Code: events.Error,
				Source: fmt.Errorf("%s: %s", c.Name, err).Error()})
		} else {
			log.Debugf("%s exited without error", c.Name)
			bus.Publish(events.Event{Code: events.ExitSuccess, Source: c.Name})
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
