package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// DecodeRaw decodes a raw interface into the target structure
func DecodeRaw(raw interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           result,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(raw)
}

// ParseCommandArgs tries to parse a command from the supported types
func ParseCommandArgs(raw interface{}) (*exec.Cmd, error) {
	switch t := raw.(type) {
	case string:
		return StrToCmd(t), nil
	}
	strArray, err := ToStringArray(raw)
	if err != nil {
		return nil, err
	}
	return ArgsToCmd(strArray), nil
}

// ToStringArray converts the given interface to a []string if possible
func ToStringArray(raw interface{}) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	switch t := raw.(type) {
	case string:
		return []string{t}, nil
	case []string:
		return t, nil
	case []interface{}:
		return interfaceToStringArray(t), nil
	default:
		return nil, fmt.Errorf("Unexpected argument type: %T", t)
	}
}

func interfaceToString(raw interface{}) string {
	switch t := raw.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func interfaceToStringArray(rawArray []interface{}) []string {
	if rawArray == nil || len(rawArray) == 0 {
		return nil
	}
	var stringArray []string
	for _, raw := range rawArray {
		stringArray = append(stringArray, interfaceToString(raw))
	}
	return stringArray
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
