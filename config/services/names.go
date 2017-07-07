package services

import (
	"fmt"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-z][a-zA-Z0-9\-]+$`)

// ValidateName checks if the service name passed as an argument
// is is alpha-numeric with dashes. This ensures compliance with both DNS
// and discovery backends.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("'name' must not be blank")
	}
	if ok := validName.MatchString(name); !ok {
		return fmt.Errorf("service names must be alphanumeric with dashes to comply with service discovery")
	}
	return nil
}
