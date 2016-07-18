package coprocesses

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

// Some magic numbers used internally by the coprocess restartLimits
const (
	haltRestarts      = -1
	unlimitedRestarts = -2
)

// Coprocess configures a process that will run alongside the main process
type Coprocess struct {
	Name     string      `mapstructure:"name"`
	Command  interface{} `mapstructure:"command"`
	Restarts interface{} `mapstructure:"restarts"`

	Args           []string
	restart        bool
	restartLimit   int
	restartsRemain int
	cmd            *exec.Cmd
	logWriters     []io.WriteCloser
}

// NewCoprocesses parses json config into an array of Coprocesses
func NewCoprocesses(raw []interface{}) ([]*Coprocess, error) {
	var coprocesses []*Coprocess
	if raw == nil {
		return coprocesses, nil
	}
	var configs []*Coprocess
	if err := utils.DecodeRaw(raw, &configs); err != nil {
		return nil, fmt.Errorf("Coprocess configuration error: %v", err)
	}
	for _, t := range configs {
		if err := parseCoprocess(t); err != nil {
			return nil, err
		}
		coprocesses = append(coprocesses, t)
	}
	return coprocesses, nil
}

func parseCoprocess(coprocess *Coprocess) error {
	args, err := utils.ToStringArray(coprocess.Command)
	if err != nil {
		return err
	}
	coprocess.Args = args
	if coprocess.Args == nil || len(coprocess.Args) == 0 {
		return fmt.Errorf("Coprocess did not provide a command")
	}
	cmd := utils.ArgsToCmd(coprocess.Args)
	coprocess.cmd = cmd

	if coprocess.Name == "" {
		coprocess.Name = strings.Join(coprocess.Args, " ")
	}

	return parseCoprocessRestarts(coprocess)
}

func parseCoprocessRestarts(coprocess *Coprocess) error {

	// defaults if omitted
	if coprocess.Restarts == nil {
		coprocess.restart = false
		coprocess.restartLimit = 0
		coprocess.restartsRemain = 0
		return nil
	}

	const msg = `Invalid 'restarts' field "%v": accepts positive integers, "unlimited" or "never"`

	switch t := coprocess.Restarts.(type) {
	case string:
		if t == "unlimited" {
			coprocess.restartLimit = unlimitedRestarts
		} else if t == "never" {
			coprocess.restartLimit = 0
		} else if i, err := strconv.Atoi(t); err == nil && i >= 0 {
			coprocess.restartLimit = i
		} else {
			return fmt.Errorf(msg, coprocess.Restarts)
		}
	case float64, int:
		// mapstructure can figure out how to decode strings into int fields
		// but doesn't try to guess and just gives us a float64 if it's got
		// a number that it's decoding to an interface{}. We'll only return
		// an error if we can't cast it to an int. This means that an end-user
		// can pass in `restarts: 1.2` and have undocumented truncation but the
		// wtf would be all on them at that point.
		if i, ok := t.(int); ok && i >= 0 {
			coprocess.restartLimit = i
		} else if i, ok := t.(float64); ok && i >= 0 {
			coprocess.restartLimit = int(i)
		} else {
			return fmt.Errorf(msg, coprocess.Restarts)
		}
	default:
		return fmt.Errorf(msg, coprocess.Restarts)
	}

	coprocess.restart = coprocess.restartLimit > 0 || coprocess.restartLimit == unlimitedRestarts
	coprocess.restartsRemain = coprocess.restartLimit
	return nil
}

// Start runs the coprocess
func (coprocess *Coprocess) Start() {
	log.Debugf("coprocess[%s].Start", coprocess.Name)
	fields := log.Fields{"process": "coprocess", "coprocess": coprocess.Name}
	stdout := utils.NewLogWriter(fields, log.InfoLevel)
	stderr := utils.NewLogWriter(fields, log.DebugLevel)
	coprocess.logWriters = []io.WriteCloser{stdout, stderr}
	defer coprocess.closeLogs()

	// always reset restartsRemain when we load the config
	coprocess.restartsRemain = coprocess.restartLimit
	for {
		if coprocess.restartLimit != unlimitedRestarts &&
			coprocess.restartsRemain <= haltRestarts {
			break
		}
		cmd := utils.ArgsToCmd(coprocess.Args)
		coprocess.cmd = cmd
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if _, err := utils.ExecuteAndWait(cmd); err != nil {
			log.Errorf("coprocess[%s] exited: %s", coprocess.Name, err)
		}
		log.Debugf("coprocess[%s] exited", coprocess.Name)
		if !coprocess.restart {
			break
		}
		coprocess.restartsRemain--
	}
}

// ShouldRestart sets whether a process from being restarted if it dies ignoring restart count
func (coprocess *Coprocess) ShouldRestart(shouldRestart bool) {
	coprocess.restart = shouldRestart
} 

// Stop kills a running coprocess
func (coprocess *Coprocess) Stop() {
	log.Debugf("coprocess[%s].Stop", coprocess.Name)
	coprocess.restartsRemain = haltRestarts
	if coprocess.cmd != nil && coprocess.cmd.Process != nil {
		log.Warnf("Killing coprocess %s: %d", coprocess.Name, coprocess.cmd.Process.Pid)
		coprocess.cmd.Process.Kill()
	}
}

func (coprocess *Coprocess) closeLogs() {
	if coprocess.logWriters == nil {
		return
	}
	for _, w := range coprocess.logWriters {
		if err := w.Close(); err != nil {
			log.Errorf("Unable to close log writer : %v", err)
		}
	}
}
