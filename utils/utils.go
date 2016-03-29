package utils

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

func ParseCommandArgs(raw json.RawMessage) (*exec.Cmd, error) {
	if raw == nil {
		return nil, nil
	}
	// Parse as a string
	var stringCmd string
	if err := json.Unmarshal(raw, &stringCmd); err == nil {
		return StrToCmd(stringCmd), nil
	}

	var arrayCmd []string
	if err := json.Unmarshal(raw, &arrayCmd); err == nil {
		return ArgsToCmd(arrayCmd), nil
	}
	return nil, errors.New("Command argument must be a string or an array")
}

func ArgsToCmd(args []string) *exec.Cmd {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return exec.Command(args[0], args[1:]...)
	}
	return exec.Command(args[0])
}

func StrToCmd(command string) *exec.Cmd {
	if command != "" {
		return ArgsToCmd(strings.Split(strings.TrimSpace(command), " "))
	}
	return nil
}

func ParseInterfaces(raw json.RawMessage) ([]string, error) {
	if raw == nil {
		return []string{}, nil
	}
	// Parse as a string
	var jsonString string
	if err := json.Unmarshal(raw, &jsonString); err == nil {
		return []string{jsonString}, nil
	}

	var jsonArray []string
	if err := json.Unmarshal(raw, &jsonArray); err == nil {
		return jsonArray, nil
	}

	return []string{}, errors.New("interfaces must be a string or an array")
}
