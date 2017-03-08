package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCoprocessConfigValidateName(t *testing.T) {
	coprocess := &CoprocessConfig{
		Exec: []string{"/usr/bin/true"},
		Name: "",
	}
	if err := coprocess.Validate(); err != nil {
		t.Fatal(err)
	}
	if coprocess.Name != "/usr/bin/true" {
		t.Fatalf("expected coprocess Name to be /usr/bin/true but got `%v`",
			coprocess.Name)
	}
}

func TestCoprocessConfigValidateCommandRequired(t *testing.T) {
	coprocess := &CoprocessConfig{}
	expected := "coprocess did not provide a command"
	if err := coprocess.Validate(); err != nil && err.Error() != expected {
		t.Fatalf("expected '%v' error but got %v", expected, err)
	}
}

func TestCoprocessConfigParsingRaw(t *testing.T) {
	expectRestarts(t, fromRaw(t, []byte(`[{"command": "true 1", "name": "me"}]`)), false, 0)
	expectRestarts(t, fromRaw(t, []byte(`[{"command": "true 2", "restarts": 1.1}]`)), true, 1)
	expectRestarts(t, fromRaw(t, []byte(`[{"command": "true 3", "restarts": 1.0}]`)), true, 1)
	expectRestarts(t, fromRaw(t, []byte(`[{"command": "true 4", "restarts": 1}]`)), true, 1)
	expectRestarts(t, fromRaw(t, []byte(`[{"command": "true 5", "restarts": "1"}]`)), true, 1)
}

func TestCoprocessConfigValidationRestarts(t *testing.T) {

	const errMsg = `accepts positive integers, "unlimited" or "never"`

	_, err := fromCoprocess(t, &CoprocessConfig{Exec: "true 1", Restarts: "invalid"})
	expectCoprocessValidationError(t, err, errMsg)
	_, err = fromCoprocess(t, &CoprocessConfig{Exec: "true 2", Restarts: "-1"})
	expectCoprocessValidationError(t, err, errMsg)
	_, err = fromCoprocess(t, &CoprocessConfig{Exec: "true 3", Restarts: -1})
	expectCoprocessValidationError(t, err, errMsg)

	service, _ := fromCoprocess(t, &CoprocessConfig{Exec: "true 4", Restarts: "unlimited"})
	expectRestarts(t, service, true, -2)
	service, _ = fromCoprocess(t, &CoprocessConfig{Exec: "true 5", Restarts: "never"})
	expectRestarts(t, service, false, 0)
	service, _ = fromCoprocess(t, &CoprocessConfig{Exec: "true 6", Restarts: 1})
	expectRestarts(t, service, true, 1)
	service, _ = fromCoprocess(t, &CoprocessConfig{Exec: "true 7", Restarts: "1"})
	expectRestarts(t, service, true, 1)
	service, _ = fromCoprocess(t, &CoprocessConfig{Exec: "true 8", Restarts: 0})
	expectRestarts(t, service, false, 0)
	service, _ = fromCoprocess(t, &CoprocessConfig{Exec: "true 9", Restarts: "0"})
	expectRestarts(t, service, false, 0)
	service, _ = fromCoprocess(t, &CoprocessConfig{Exec: "true 10"})
	expectRestarts(t, service, false, 0)
}

// test helper functions

func fromRaw(t *testing.T, j json.RawMessage) *ServiceConfig {
	var raw []interface{}
	if err := json.Unmarshal(j, &raw); err != nil {
		t.Fatalf("unexpected error decoding JSON:\n%s\n%v", j, err)
	} else if coprocesses, err := NewCoprocessConfigs(raw); err != nil {
		t.Fatalf("expected no errors from %v but got %v", raw, err)
	} else {
		return coprocesses[0]
	}
	return nil
}

func fromCoprocess(t *testing.T, co *CoprocessConfig) (*ServiceConfig, error) {
	if err := co.Validate(); err != nil {
		t.Fatalf("unexpected error in coprocess.Validate: %v", err)
	}
	service := co.ToServiceConfig()
	return service, service.Validate(nil)
}

func expectCoprocessValidationError(t *testing.T, err error, expected string) {
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error '%s' but got: %v", expected, err)
	}
}
