package utils

import (
	"fmt"
	"strconv"
	"time"
)

// ParseDuration parses the given duration with multiple type support
// int (defaults to seconds)
// string with units
// string without units (default to seconds)
func ParseDuration(duration interface{}) (time.Duration, error) {
	switch t := duration.(type) {
	default:
		return time.Second, fmt.Errorf("unexpected duration of type %T\n", t)
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
