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
func TestValidConfigJobs(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig(testJSON)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(cfg.Jobs) != 8 {
		t.Fatalf("expected 8 services but got %v", cfg.Jobs)
	}
	job0 := cfg.Jobs[0]
	assertEqual(t, job0.Name, "serviceA", "expected '%v' for job0.Name but got '%v'")
	assertEqual(t, job0.Port, 8080, "expected '%v' for job0.Port but got '%v'")
	assertEqual(t, job0.Exec, "/bin/serviceA", "expected '%v' for job0.Exec but got '%v'")
	assertEqual(t, job0.Tags, []string{"tag1", "tag2"}, "expected '%v' for job0.Tags but got '%v'")
	assertEqual(t, job0.Restarts, nil, "expected '%v' for job1.Restarts but got '%v'")

	job1 := cfg.Jobs[1]
	assertEqual(t, job1.Name, "serviceB", "expected '%v' for job1.Name but got '%v'")
	assertEqual(t, job1.Port, 5000, "expected '%v' for job1.Port but got '%v'")
	assertEqual(t, len(job1.Tags), 0, "expected '%v' for len(job1.Tags) but got '%v'")
	assertEqual(t, job1.Exec, []interface{}{"/bin/serviceB", "B"}, "expected '%v' for job1.Exec but got '%v'")
	assertEqual(t, job1.Restarts, nil, "expected '%v' for job1.Restarts but got '%v'")

	job2 := cfg.Jobs[2]
	assertEqual(t, job2.Name, "coprocessC", "expected '%v' for job2.Name but got '%v'")
	assertEqual(t, job2.Port, 0, "expected '%v' for job2.Port but got '%v'")
	assertEqual(t, job2.Frequency, "", "expected '%v' for job2.Frequency but got '%v'")
	assertEqual(t, job2.Restarts, "unlimited", "expected '%v' for job2.Restarts but got '%v'")

	job3 := cfg.Jobs[3]
	assertEqual(t, job3.Name, "taskD", "expected '%v' for job3.Name but got '%v'")
	assertEqual(t, job3.Port, 0, "expected '%v' for job3.Port but got '%v'")
	assertEqual(t, job3.Frequency, "1s", "expected '%v' for job3.Frequency but got '%v'")
	assertEqual(t, job3.Restarts, nil, "expected '%v' for job3.Restarts but got '%v'")

	job4 := cfg.Jobs[4]
	assertEqual(t, job4.Name, "serviceA.preStart", "expected '%v' for job4.Name but got '%v'")
	assertEqual(t, job4.Port, 0, "expected '%v' for job4.Port but got '%v'")
	assertEqual(t, job4.Frequency, "", "expected '%v' for job4.Frequency but got '%v'")
	assertEqual(t, job4.Restarts, nil, "expected '%v' for job4.Restarts but got '%v'")

	job5 := cfg.Jobs[5]
	assertEqual(t, job5.Name, "serviceA.preStop", "expected '%v' for job5.Name but got '%v'")
	assertEqual(t, job5.Port, 0, "expected '%v' for job5.Port but got '%v'")
	assertEqual(t, job5.Frequency, "", "expected '%v' for job5.Frequency but got '%v'")
	assertEqual(t, job5.Restarts, nil, "expected '%v' for job5.Restarts but got '%v'")

	job6 := cfg.Jobs[6]
	assertEqual(t, job6.Name, "serviceA.postStop", "expected '%v' for job6.Name but got '%v'")
	assertEqual(t, job6.Port, 0, "expected '%v' for job6.Port but got '%v'")
	assertEqual(t, job6.Frequency, "", "expected '%v' for job6.Frequency but got '%v'")
	assertEqual(t, job6.Restarts, nil, "expected '%v' for job6.Restarts but got '%v'")

	job7 := cfg.Jobs[7]
	assertEqual(t, job7.Name, "containerpilot", "expected '%v' for job7.Name but got '%v'")
	assertEqual(t, job7.Port, 9000, "expected '%v' for job7.Port but got '%v'")
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
