package services

import (
	"encoding/json"
	"os/exec"
	"reflect"
	"testing"
	"utils"
)

func TestHealthCheck(t *testing.T) {
	cmd1 := utils.StrToCmd("./testdata/test.sh doStuff --debug")
	service := &ServiceConfig{
		healthCheckCmd: cmd1,
	}
	if _, err := service.CheckHealth(); err != nil {
		t.Errorf("Unexpected error CheckHealth: %s", err)
	}
	// Ensure we can run it more than once
	if _, err := service.CheckHealth(); err != nil {
		t.Errorf("Unexpected error CheckHealth (x2): %s", err)
	}
}

type TestFragmentServices struct {
	Services []ServiceConfig
}

func TestServiceParse(t *testing.T) {
	jsonFragment := []byte(`{"services": [
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "eth0",
  "health": "/bin/to/healthcheck/for/service/A.sh",
  "poll": 30,
  "ttl": 19,
  "tags": ["tag1","tag2"]
},
{
  "name": "serviceB",
  "port": 5000,
  "interfaces": ["ethwe","eth0"],
  "health": "/bin/to/healthcheck/for/service/B.sh",
  "poll": 30,
  "ttl": 103
}
]}`)
	serviceFragment := &TestFragmentServices{}
	if err := json.Unmarshal(jsonFragment, serviceFragment); err != nil {
		t.Fatalf("Could not parse service JSON: %s", err)
	} else {
		services := serviceFragment.Services
		service1 := services[0]
		if err := service1.Parse(nil); err != nil {
			t.Fatalf("Could not parse services: %s", err)
		} else {
			validateCommandParsed(t, "health", service1.healthCheckCmd,
				[]string{"/bin/to/healthcheck/for/service/A.sh"})
		}
		service2 := services[1]
		if err := service2.Parse(nil); err != nil {
			t.Fatalf("Could not parse services: %s", err)
		} else {
			validateCommandParsed(t, "health", service2.healthCheckCmd,
				[]string{"/bin/to/healthcheck/for/service/B.sh"})
		}
	}
}

// ------------------------------------------
// test helpers

func validateCommandParsed(t *testing.T, name string, parsed *exec.Cmd, expected []string) {
	if expected == nil {
		if parsed != nil {
			t.Errorf("%s has Cmd, but expected nil", name)
		}
		return
	}
	if parsed == nil {
		t.Errorf("%s not configured", name)
	}
	if parsed.Path != expected[0] {
		t.Errorf("%s path not configured: %s != %s", name, parsed.Path, expected[0])
	}
	if !reflect.DeepEqual(parsed.Args, expected) {
		t.Errorf("%s arguments not configured: %s != %s", name, parsed.Args, expected)
	}
}
