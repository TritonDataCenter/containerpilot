package utils

import (
	"fmt"
	"strconv"
	"time"
)

// GetTimeout converts a properly formatted string to a Duration,
// returning an error if the Duration can't be parsed
func GetTimeout(timeoutFmt string) (time.Duration, error) {
	if timeoutFmt != "" {
		timeout, err := ParseDuration(timeoutFmt)
		if err != nil {
			return time.Duration(0), err
		}
		return timeout, nil
	}
	return time.Duration(0), nil
}

// ParseDuration parses the given duration with multiple type support
// int (defaults to seconds)
// string with units
// string without units (default to seconds)
func ParseDuration(duration interface{}) (time.Duration, error) {
	switch t := duration.(type) {
	default:
		return time.Second, fmt.Errorf("unexpected duration of type %T", t)
	case int64:
		return time.Duration(t) * time.Second, nil
	case int32:
		return time.Duration(t) * time.Second, nil
	case int16:
		return time.Duration(t) * time.Second, nil
	case int8:
		return time.Duration(t) * time.Second, nil
	case int:
		return time.Duration(t) * time.Second, nil
	case uint64:
		return time.Duration(t) * time.Second, nil
	case uint32:
		return time.Duration(t) * time.Second, nil
	case uint16:
		return time.Duration(t) * time.Second, nil
	case uint8:
		return time.Duration(t) * time.Second, nil
	case uint:
		return time.Duration(t) * time.Second, nil
	case string:
		if i, err := strconv.Atoi(t); err == nil {
			return time.ParseDuration(fmt.Sprintf("%ds", i))
		}
		return time.ParseDuration(t)
	}
}
