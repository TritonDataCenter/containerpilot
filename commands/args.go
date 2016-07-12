package commands

import (
	"os/exec"
	"strings"

	"github.com/joyent/containerpilot/utils"
)

// ParseCommandArgs tries to parse a command from the supported types
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
