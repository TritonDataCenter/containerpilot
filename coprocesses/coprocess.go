package coprocesses

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/commands"
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

	restart        bool
	restartLimit   int
	restartsRemain int
	cmd            *commands.Command
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
	if coprocess.Command == nil {
		return fmt.Errorf("Coprocess did not provide a command")
	}
	cmd, err := commands.NewCommand(coprocess.Command, "0")
	if err != nil {
		return fmt.Errorf("Could not parse `coprocess` command %s: %s",
			coprocess.Name, err)
	}
	if coprocess.Name == "" {
		args := append([]string{cmd.Exec}, cmd.Args...)
		coprocess.Name = strings.Join(args, " ")
	}
	cmd.Name = fmt.Sprintf("coprocess[%s]", coprocess.Name)
	coprocess.cmd = cmd
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

	coprocess.restart = (coprocess.restartLimit > 0 ||
		coprocess.restartLimit == unlimitedRestarts)
	coprocess.restartsRemain = coprocess.restartLimit
	return nil
}

// Start runs the coprocess
func (c *Coprocess) Start() {
	log.Debugf("coprocess[%s].Start", c.Name)
	fields := log.Fields{"process": "coprocess", "coprocess": c.Name}

	// always reset restartsRemain when we load the config
	c.restartsRemain = c.restartLimit
	for {
		if c.restartLimit != unlimitedRestarts &&
			c.restartsRemain <= haltRestarts {
			break
		}
		if code, err := commands.RunAndWait(c.cmd, fields); err != nil {
			log.Errorf("coprocess[%s] exited (%s): %s", c.Name, code, err)
		}
		log.Debugf("coprocess[%s] exited", c.Name)
		if !c.restart {
			break
		}
		c.restartsRemain--
	}
}

// Stop kills a running coprocess
func (c *Coprocess) Stop() {
	log.Debugf("coprocess[%s].Stop", c.Name)
	c.restartsRemain = haltRestarts
	c.restartLimit = haltRestarts
	c.restart = false
	c.cmd.Kill()
}
