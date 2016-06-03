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
			t.Errorf("Expected no error for name `%v` but got %v", name, err)
		}
	}

	// TODO: in a future version of ContainerPilot invalid service names
	//       will be an error rather than a warning. When that happens,
	//       uncomment this test.
	//
	// var invalidNames = []string{
	// 	"my_service",
	// 	"-my-service",
	// 	"my%service",
	// }
	// for _, name := range invalidNames {
	// 	if err := ValidateServiceName(name); err == nil {
	// 		t.Errorf("Expected error for name `%v` but got nil", name)
	// 	}
	// }
}
