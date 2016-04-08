package services

import (
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
	jsonFragment := []byte(`[
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
]`)
	if services, err := NewServices(jsonFragment, nil); err != nil {
		t.Fatalf("Could not parse service JSON: %s", err)
	} else {
		validateCommandParsed(t, "health", services[0].healthCheckCmd,
			[]string{"/bin/to/healthcheck/for/service/A.sh"})
		validateCommandParsed(t, "health", services[1].healthCheckCmd,
			[]string{"/bin/to/healthcheck/for/service/B.sh"})
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
