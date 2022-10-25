package commands

import (
	"errors"
	"strings"

	"github.com/tritondatacenter/containerpilot/config/decode"
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
		args, err = decode.ToStrings(raw)
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
