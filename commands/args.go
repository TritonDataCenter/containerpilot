package commands

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/joyent/containerpilot/utils"
)

// ParseArgs parses the executable and its arguments from supported
// types.
func ParseArgs(raw interface{}) (executable string, args []string, err error) {
	switch t := raw.(type) {
	case string:
		if t != "" {
			args = strings.Split(strings.TrimSpace(t), " ")
		}
	default:
		args, err = utils.ToStringArray(raw)
	}
	if len(args) == 0 {
		err = errors.New("received zero-length argument")
	} else if len(args) == 1 {
		executable = args[0]
		args = nil
	} else {
		executable = args[0]
		args = args[1:]
	}
	return executable, args, err
}

// ParseCommandArgs tries to parse a command from the supported types
// TODO: remove once we've finished refactor
func ParseCommandArgs(raw interface{}) (*exec.Cmd, error) {
	switch t := raw.(type) {
	case string:
		return StrToCmd(t), nil
	}
	strArray, err := utils.ToStringArray(raw)
	if err != nil {
		return nil, err
	}
	return ArgsToCmd(strArray), nil
}

// ArgsToCommand creates a command from a list of arguments
// TODO: rename this to ArgsToCmd once we've finished refactor
func ArgsToCommand(executable string, args []string) *exec.Cmd {
	if len(args) == 0 {
		return exec.Command(executable)
	}
	return exec.Command(executable, args...)
}

// ArgsToCmd creates a command from a list of arguments
func ArgsToCmd(args []string) *exec.Cmd {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return exec.Command(args[0], args[1:]...)
	}
	return exec.Command(args[0])
}

// StrToCmd creates a command from a string, triming whitespace
func StrToCmd(command string) *exec.Cmd {
	if command != "" {
		return ArgsToCmd(strings.Split(strings.TrimSpace(command), " "))
	}
	return nil
}
