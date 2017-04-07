package utils

import (
	"testing"
)

func TestValidateServiceName(t *testing.T) {

	var validNames = []string{
		"myService",
		"my-service",
		"my-service-123",
	}
	for _, name := range validNames {
		if err := ValidateServiceName(name); err != nil {
			t.Errorf("expected no error for name '%v' but got %v", name, err)
		}
	}

	var invalidNames = []string{
		"my_service",
		"-my-service",
		"my%service",
	}
	for _, name := range invalidNames {
		if err := ValidateServiceName(name); err == nil {
			t.Errorf("expected error for name '%v' but got nil", name)
		}
	}
}
