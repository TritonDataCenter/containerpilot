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
func (self MultiFlag) String() string {
	return fmt.Sprintf("%v", self.Values)
}

// Set satisfies the flag.Value interface by creating a map of all unique CLI
// flag values.
func (self *MultiFlag) Set(value string) error {
	if self.Len() == 0 {
		self.Values = make(map[string]string, 1)
	}
	pair := strings.Split(value, "=")
	key, val := strings.Join(pair[0:1], ""), strings.Join(pair[1:2], "")
	self.Values[key] = val
	return nil
}

// Len is the length of the slice of values for this MultiFlag.
func (self MultiFlag) Len() int {
	return len(self.Values)
}
