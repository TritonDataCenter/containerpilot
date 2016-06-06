package utils

import (
	"fmt"
	"regexp"

	log "github.com/Sirupsen/logrus"
)

var validName = regexp.MustCompile(`^[a-z][a-zA-Z0-9\-]+$`)

// ValidateServiceName checks if the service name passed as an argument
// is is alpha-numeric with dashes. This ensures compliance with both DNS
// and discovery backends.
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("`name` must not be blank")
	}
	if ok := validName.MatchString(name); !ok {
		log.Warnf("Deprecation warning: service names must be alpha-numeric with dashes. In a future version of ContainerPilot this will be an error.")
	}
	return nil
}
