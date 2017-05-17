package core

import (
	"fmt"
	"strings"
)

// MultiFlag provides a custom CLI flag that stores its unique values into a
// simple map.
type MultiFlag struct {
	Values map[string]string
}

// String satisfies the flag.Value interface by joining together the flag values
// map into a single String.
func (f MultiFlag) String() string {
	return fmt.Sprintf("%v", f.Values)
}

// Set satisfies the flag.Value interface by creating a map of all unique CLI
// flag values.
func (f *MultiFlag) Set(value string) error {
	if f.Len() == 0 {
		f.Values = make(map[string]string, 1)
	}
	pair := strings.Split(value, "=")
	if len(pair) < 2 {
		return fmt.Errorf(
			"flag value '%v' was not in the format 'key=val'", value)
	}
	key, val := strings.Join(pair[0:1], ""), strings.Join(pair[1:2], "")
	f.Values[key] = val
	return nil
}

// Len is the length of the slice of values for this MultiFlag.
func (f MultiFlag) Len() int {
	return len(f.Values)
}
