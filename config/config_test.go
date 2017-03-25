package config

import (
	"os"
	"reflect"
	"testing"
)

/*
This is mostly a giant suite of smoke tests, as the detailed tests of validation
are all in the individual package tests. Here we'll make sure all components
come together as we expect and also check things like env var interpolation.
*/

var testJSON = `{
	"consul": "consul:8500",
	"stopTimeout": 5,
	"services": [
			{
					"name": "serviceA",
					"port": 8080,
					"interfaces": "inet",
					"exec": "/bin/serviceA",
					"preStart": "/bin/to/preStart.sh arg1 arg2",
					"preStop": ["/bin/to/preStop.sh","arg1","arg2"],
					"postStop": ["/bin/to/postStop.sh"],
					"health": "/bin/to/healthcheck/for/service/A.sh",
					"poll": 30,
					"ttl": "19",
					"tags": ["tag1","tag2"]
			},
			{
					"name": "serviceB",
					"port": 5000,
					"interfaces": ["ethwe","eth0", "inet"],
					"exec": ["/bin/serviceB", "B"],
					"health": ["/bin/to/healthcheck/for/service/B.sh", "B"],
					"timeout": "2s",
					"poll": 20,
					"ttl": "103"
			},
			{
					"name": "coprocessC",
					"exec": "/bin/coprocessC",
					"restarts": "unlimited"
			},
			{
					"name": "taskD",
					"exec": "/bin/taskD",
					"frequency": "1s"
			}
	],
	"backends": [
			{
					"name": "upstreamA",
					"poll": 11,
					"onChange": "/bin/to/onChangeEvent/for/upstream/A.sh {{.TEST}}",
					"tag": "dev"
			},
			{
					"name": "upstreamB",
					"poll": 79,
					"onChange": "/bin/to/onChangeEvent/for/upstream/B.sh {{.ENV_NOT_FOUND}}"
			}
	],
	"telemetry": {
		"port": 9000,
		"interfaces": ["inet"],
		"tags": ["dev"],
		"sensors": [
			{
				"namespace": "org",
				"subsystem": "app",
				"name": "zed",
				"help": "gauge of zeds in org app",
				"type": "gauge",
				"poll": 10,
				"check": "/bin/sensorZ",
				"timeout": "5s"
			}
		]
	}
}
`

func assertEqual(t *testing.T, got, expected interface{}, msg string) {
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(msg, expected, got)
	}
}

// checks.Config
func TestValidConfigHealthChecks(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig(testJSON)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(cfg.Checks) != 2 {
		t.Fatalf("expected 2 checks but got %+v", cfg.Checks)
	}
	check0 := cfg.Checks[0]
	check1 := cfg.Checks[1]
	assertEqual(t, check0.Name, "serviceA.check", "expected '%v' for check0.Name, but got '%v'")
	assertEqual(t, check0.Poll, 30, "expected '%v' for check0.Poll, but got '%v'")
	assertEqual(t, check0.Timeout, "30s", "expected '%v' for check0.Timeout, but got '%v'")
	assertEqual(t, check1.Name, "serviceB.check", "expected '%v' for check1.Name, but got '%v'")
	assertEqual(t, check1.Poll, 20, "expected '%v' for check1.Poll, but got '%v'")
	assertEqual(t, check1.Timeout, "2s", "expected '%v' for check1.Timeout, but got '%v'")
}

// services.Config
func TestValidConfigServices(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig(testJSON)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(cfg.Services) != 8 {
		t.Fatalf("expected 8 services but got %v", cfg.Services)
	}
	svc0 := cfg.Services[0]
	assertEqual(t, svc0.Name, "serviceA", "expected '%v' for svc0.Name but got '%v'")
	assertEqual(t, svc0.Port, 8080, "expected '%v' for svc0.Port but got '%v'")
	assertEqual(t, svc0.Exec, "/bin/serviceA", "expected '%v' for svc0.Exec but got '%v'")
	assertEqual(t, svc0.Tags, []string{"tag1", "tag2"}, "expected '%v' for svc0.Tags but got '%v'")
	assertEqual(t, svc0.Restarts, nil, "expected '%v' for svc1.Restarts but got '%v'")

	svc1 := cfg.Services[1]
	assertEqual(t, svc1.Name, "serviceB", "expected '%v' for svc1.Name but got '%v'")
	assertEqual(t, svc1.Port, 5000, "expected '%v' for svc1.Port but got '%v'")
	assertEqual(t, len(svc1.Tags), 0, "expected '%v' for len(svc1.Tags) but got '%v'")
	assertEqual(t, svc1.Exec, []interface{}{"/bin/serviceB", "B"}, "expected '%v' for svc1.Exec but got '%v'")
	assertEqual(t, svc1.Restarts, nil, "expected '%v' for svc1.Restarts but got '%v'")

	svc2 := cfg.Services[2]
	assertEqual(t, svc2.Name, "coprocessC", "expected '%v' for svc2.Name but got '%v'")
	assertEqual(t, svc2.Port, 0, "expected '%v' for svc2.Port but got '%v'")
	assertEqual(t, svc2.Frequency, "", "expected '%v' for svc2.Frequency but got '%v'")
	assertEqual(t, svc2.Restarts, "unlimited", "expected '%v' for svc2.Restarts but got '%v'")

	svc3 := cfg.Services[3]
	assertEqual(t, svc3.Name, "taskD", "expected '%v' for svc3.Name but got '%v'")
	assertEqual(t, svc3.Port, 0, "expected '%v' for svc3.Port but got '%v'")
	assertEqual(t, svc3.Frequency, "1s", "expected '%v' for svc3.Frequency but got '%v'")
	assertEqual(t, svc3.Restarts, nil, "expected '%v' for svc3.Restarts but got '%v'")

	svc4 := cfg.Services[4]
	assertEqual(t, svc4.Name, "serviceA.preStart", "expected '%v' for svc4.Name but got '%v'")
	assertEqual(t, svc4.Port, 0, "expected '%v' for svc4.Port but got '%v'")
	assertEqual(t, svc4.Frequency, "", "expected '%v' for svc4.Frequency but got '%v'")
	assertEqual(t, svc4.Restarts, nil, "expected '%v' for svc4.Restarts but got '%v'")

	svc5 := cfg.Services[5]
	assertEqual(t, svc5.Name, "serviceA.preStop", "expected '%v' for svc5.Name but got '%v'")
	assertEqual(t, svc5.Port, 0, "expected '%v' for svc5.Port but got '%v'")
	assertEqual(t, svc5.Frequency, "", "expected '%v' for svc5.Frequency but got '%v'")
	assertEqual(t, svc5.Restarts, nil, "expected '%v' for svc5.Restarts but got '%v'")

	svc6 := cfg.Services[6]
	assertEqual(t, svc6.Name, "serviceA.postStop", "expected '%v' for svc6.Name but got '%v'")
	assertEqual(t, svc6.Port, 0, "expected '%v' for svc6.Port but got '%v'")
	assertEqual(t, svc6.Frequency, "", "expected '%v' for svc6.Frequency but got '%v'")
	assertEqual(t, svc6.Restarts, nil, "expected '%v' for svc6.Restarts but got '%v'")

	svc7 := cfg.Services[7]
	assertEqual(t, svc7.Name, "containerpilot", "expected '%v' for svc7.Name but got '%v'")
	assertEqual(t, svc7.Port, 9000, "expected '%v' for svc7.Port but got '%v'")
}

// telemetry.Config
func TestValidConfigTelemetry(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig(testJSON)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	telem := cfg.Telemetry
	sensor0 := telem.SensorConfigs[0]
	assertEqual(t, telem.Port, 9000, "expected '%v' for telem.Port but got '%v")
	assertEqual(t, telem.Tags, []string{"dev"}, "expected '%v' for telem.Tags but got '%v")
	assertEqual(t, sensor0.Timeout, "5s", "expected '%v' for sensor0.Timeout but got '%v")
	assertEqual(t, sensor0.Poll, 10, "expected '%v' for sensor0.Poll but got '%v")
}

// watches.Config
func TestValidConfigWatches(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig(testJSON)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(cfg.Watches) != 2 {
		t.Fatalf("expected 2 watches but got %v", cfg.Watches)
	}
	watch0 := cfg.Watches[0]
	watch1 := cfg.Watches[1]
	assertEqual(t, watch0.Name, "upstreamA.watch", "expected '%v' for Name, but got '%v'")
	assertEqual(t, watch0.Poll, 11, "expected '%v' for Poll, but got '%v'")
	assertEqual(t, watch0.Tag, "dev", "expected '%v' for Tag, but got '%v'")
	assertEqual(t, watch1.Name, "upstreamB.watch", "expected '%v' for Name, but got '%v'")
	assertEqual(t, watch1.Poll, 79, "expected '%v' for Poll, but got '%v'")
	assertEqual(t, watch1.Tag, "", "expected '%v' for Tag, but got '%v'")

}

// checks.Config
func TestValidConfigControl(t *testing.T) {
	cfg, err := LoadConfig(testJSON)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	assertEqual(t, cfg.Control.SocketPath, "/var/run/containerpilot.socket", "expected '%v' for control.socket, but got '%v'")
}

func TestCustomConfigControl(t *testing.T) {
	var testJSONWithSocket = `{
	"control": {
		"socket": "/var/run/cp3-test.sock"
	},
	"consul": "consul:8500"
}`

	cfg, err := LoadConfig(testJSONWithSocket)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	assertEqual(t, cfg.Control.SocketPath, "/var/run/cp3-test.sock", "expected '%v' for control.socket, but got '%v'")
}
