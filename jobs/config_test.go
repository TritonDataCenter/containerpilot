package jobs

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
	"github.com/joyent/containerpilot/tests/mocks"
)

var noop = &mocks.NoopDiscoveryBackend{}

func TestJobConfigHappyPath(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))

	jobs, err := NewConfigs(testCfg, noop)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(jobs) != 7 {
		t.Fatalf("expected 7 jobs but got %v", jobs)
	}
	job0 := jobs[0]
	assert.Equal(t, job0.Name, "serviceA", "expected '%v' for job0.Name but got '%v'")
	assert.Equal(t, job0.Port, 8080, "expected '%v' for job0.Port but got '%v'")
	assert.Equal(t, job0.Exec, "/bin/serviceA", "expected '%v' for job0.Exec but got '%v'")
	assert.Equal(t, job0.Tags, []string{"tag1", "tag2"}, "expected '%v' for job0.Tags but got '%v'")
	assert.Equal(t, job0.Restarts, nil, "expected '%v' for job1.Restarts but got '%v'")

	job1 := jobs[1]
	assert.Equal(t, job1.Name, "serviceB", "expected '%v' for job1.Name but got '%v'")
	assert.Equal(t, job1.Port, 5000, "expected '%v' for job1.Port but got '%v'")
	assert.Equal(t, len(job1.Tags), 0, "expected '%v' for len(job1.Tags) but got '%v'")
	assert.Equal(t, job1.Exec, []interface{}{"/bin/serviceB", "B"}, "expected '%v' for job1.Exec but got '%v'")
	assert.Equal(t, job1.Restarts, nil, "expected '%v' for job1.Restarts but got '%v'")

	job2 := jobs[2]
	assert.Equal(t, job2.Name, "coprocessC", "expected '%v' for job2.Name but got '%v'")
	assert.Equal(t, job2.Port, 0, "expected '%v' for job2.Port but got '%v'")
	assert.Equal(t, job2.Frequency, "", "expected '%v' for job2.Frequency but got '%v'")
	assert.Equal(t, job2.Restarts, "unlimited", "expected '%v' for job2.Restarts but got '%v'")

	job3 := jobs[3]
	assert.Equal(t, job3.Name, "taskD", "expected '%v' for job3.Name but got '%v'")
	assert.Equal(t, job3.Port, 0, "expected '%v' for job3.Port but got '%v'")
	assert.Equal(t, job3.Frequency, "1s", "expected '%v' for job3.Frequency but got '%v'")
	assert.Equal(t, job3.Restarts, nil, "expected '%v' for job3.Restarts but got '%v'")

	job4 := jobs[4]
	assert.Equal(t, job4.Name, "preStart", "expected '%v' for job4.Name but got '%v'")
	assert.Equal(t, job4.Port, 0, "expected '%v' for job4.Port but got '%v'")
	assert.Equal(t, job4.Frequency, "", "expected '%v' for job4.Frequency but got '%v'")
	assert.Equal(t, job4.Restarts, nil, "expected '%v' for job4.Restarts but got '%v'")

	job5 := jobs[5]
	assert.Equal(t, job5.Name, "preStop", "expected '%v' for job5.Name but got '%v'")
	assert.Equal(t, job5.Port, 0, "expected '%v' for job5.Port but got '%v'")
	assert.Equal(t, job5.Frequency, "", "expected '%v' for job5.Frequency but got '%v'")
	assert.Equal(t, job5.Restarts, nil, "expected '%v' for job5.Restarts but got '%v'")

	job6 := jobs[6]
	assert.Equal(t, job6.Name, "postStop", "expected '%v' for job6.Name but got '%v'")
	assert.Equal(t, job6.Port, 0, "expected '%v' for job6.Port but got '%v'")
	assert.Equal(t, job6.Frequency, "", "expected '%v' for job6.Frequency but got '%v'")
	assert.Equal(t, job6.Restarts, nil, "expected '%v' for job6.Restarts but got '%v'")
}

func TestJobConfigValidateName(t *testing.T) {

	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": ""}]`), noop)
	assert.Error(t, err, "`name` must not be blank")

	cfg, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": "", "exec": "myexec"}]`), noop)
	assert.Error(t, err, "`name` must not be blank")

	// no name permitted only if no discovery backend assigned
	cfg, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "", "exec": "myexec"}]`), nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg[0].Name, "myexec", "expected '%v' for cfg.Name got '%v'")
}

func TestJobConfigValidateDiscovery(t *testing.T) {
	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "port": 80}]`), noop)
	assert.Error(t, err, "`poll` must be > 0 in job `myName` when `port` is set")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "port": 80, "poll": 1}]`), noop)
	assert.Error(t, err, "`ttl` must be > 0 in job `myName` when `port` is set")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "poll": 1, "ttl": 1}]`), noop)
	assert.Error(t, err,
		"`heartbeat` and `ttl` may not be set in job `myName` if `port` is not set")

	// no health check shouldn't return an error
	raw := tests.DecodeRawToSlice(`[{"name": "myName", "poll": 1, "ttl": 1, "port": 80}]`)
	if _, err = NewConfigs(raw, noop); err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
}

func TestJobsConsulExtrasEnableTagOverride(t *testing.T) {
	testCfg, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	jobs, err := NewConfigs(tests.DecodeRawToSlice(string(testCfg)), nil)
	if err != nil {
		t.Fatalf("could not parse job JSON: %s", err)
	}
	if jobs[0].definition.ConsulExtras.EnableTagOverride != true {
		t.Errorf("ConsulExtras should have had EnableTagOverride set to true.")
	}
}

func TestInvalidJobsConsulExtrasEnableTagOverride(t *testing.T) {
	testCfg, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	_, err := NewConfigs(tests.DecodeRawToSlice(string(testCfg)), nil)
	if err == nil {
		t.Errorf("ConsulExtras should have thrown error about EnableTagOverride being a string.")
	}
}

func TestJobsConsulExtrasDeregisterCriticalServiceAfter(t *testing.T) {
	testCfg, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	jobs, err := NewConfigs(tests.DecodeRawToSlice(string(testCfg)), nil)
	if err != nil {
		t.Fatalf("could not parse job JSON: %s", err)
	}
	if jobs[0].definition.ConsulExtras.DeregisterCriticalServiceAfter != "40m" {
		t.Errorf("ConsulExtras should have had DeregisterCriticalServiceAfter set to '40m'.")
	}
}

func TestInvalidJobsConsulExtrasDeregisterCriticalServiceAfter(t *testing.T) {
	testCfg, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	_, err := NewConfigs(tests.DecodeRawToSlice(string(testCfg)), nil)
	if err == nil {
		t.Errorf("error should have been generated for duration 'nope'.")
	}
}

func TestJobConfigValidateFrequency(t *testing.T) {
	expectErr := func(test, errMsg string) {
		testCfg := tests.DecodeRawToSlice(test)
		_, err := NewConfigs(testCfg, nil)
		assert.Error(t, err, errMsg)
	}
	expectErr(`[{"exec": "/bin/taskA", "frequency": "-1s", "execTimeout": "1s"}]`,
		"frequency '-1s' cannot be less than 1ms")

	expectErr(`[{"exec": "/bin/taskB", "frequency": "1ns", "execTimeout": "1s"}]`,
		"frequency '1ns' cannot be less than 1ms")

	expectErr(`[{"exec": "/bin/taskC", "frequency": "1ms", "execTimeout": "-1ms"}]`,
		"timeout '-1ms' cannot be less than 1ms")

	expectErr(`[{"exec": "/bin/taskD", "frequency": "1ms", "execTimeout": "1ns"}]`,
		"timeout '1ns' cannot be less than 1ms")

	expectErr(`[{"exec": "/bin/taskD", "frequency": "xx", "execTimeout": "1ns"}]`,
		"unable to parse frequency 'xx': time: invalid duration xx")

	testCfg := tests.DecodeRawToSlice(`[{"exec": "/bin/taskE", "frequency": "1ms"}]`)
	job, _ := NewConfigs(testCfg, nil)
	assert.Equal(t, job[0].execTimeout, job[0].freqInterval,
		"expected execTimeout '%v' to equal frequency '%v'")
}

func TestJobConfigValidateExec(t *testing.T) {

	testCfg := tests.DecodeRawToSlice(`[
	{
		"name": "serviceA",
		"exec": ["/bin/serviceA", "A1", "A2"],
		"health": ["/bin/to/healthcheck/for/service/A.sh", "A1", "A2"],
		"execTimeout": "1ms"
	}]`)
	cfg, err := NewConfigs(testCfg, noop)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg[0].exec.Exec, "/bin/serviceA",
		"expected %v for serviceA.exec.Exec got %v")
	assert.Equal(t, cfg[0].exec.Args, []string{"A1", "A2"},
		"expected %v for serviceA.exec.Args got %v")
	assert.Equal(t, cfg[0].execTimeout, time.Duration(time.Millisecond),
		"expected %v for serviceA.execTimeout got %v")

	testCfg = tests.DecodeRawToSlice(`[
	{
		"name": "serviceB",
		"exec": "/bin/serviceB B1 B2",
		"health": "/bin/to/healthcheck/for/service/B.sh B1 B2",
		"execTimeout": "1ms"
	}]`)
	cfg, err = NewConfigs(testCfg, noop)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg[0].exec.Exec, "/bin/serviceB",
		"expected %v for serviceB.exec.Exec got %v")
	assert.Equal(t, cfg[0].exec.Args, []string{"B1", "B2"},
		"expected %v for serviceB.exec.Args got %v")
	assert.Equal(t, cfg[0].execTimeout, time.Duration(time.Millisecond),
		"expected %v for serviceB.execTimeout got %v")

	testCfg = tests.DecodeRawToSlice(`[
	{
		"name": "serviceC",
		"exec": "/bin/serviceC C1 C2",
		"execTimeout": "xx"
	}]`)
	_, err = NewConfigs(testCfg, noop)
	expected := "could not parse `timeout` for job serviceC: time: invalid duration xx"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected '%s', got '%v'", expected, err)
	}

	testCfg = tests.DecodeRawToSlice(`[
	{
		"name": "serviceD",
		"exec": ""
	}]`)
	_, err = NewConfigs(testCfg, noop)
	expected = "could not parse `exec` for job serviceD: received zero-length argument"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected '%s', got '%v'", expected, err)
	}

}

func TestJobConfigValidateRestarts(t *testing.T) {

	expectErr := func(test, val string) {
		errMsg := fmt.Sprintf(`invalid 'restarts' field "%v": accepts positive integers, "unlimited", or "never"`, val)
		testCfg := tests.DecodeRawToSlice(test)
		_, err := NewConfigs(testCfg, nil)
		assert.Error(t, err, errMsg)
	}
	expectErr(`[{"exec": "/bin/coprocessA", "restarts": "invalid"}]`, "invalid")
	expectErr(`[{"exec": "/bin/coprocessB", "restarts": "-1"}]`, "-1")
	expectErr(`[{"exec": "/bin/coprocessC", "restarts": -1 }]`, "-1")

	testCfg := tests.DecodeRawToSlice(`[
	{ "exec": "/bin/coprocessD", "restarts": "unlimited" },
	{ "exec": "/bin/coprocessE", "restarts": "never" },
	{ "exec": "/bin/coprocessF", "restarts": 1 },
	{ "exec": "/bin/coprocessG", "restarts": "1" },
	{ "exec": "/bin/coprocessH", "restarts": 0 },
	{ "exec": "/bin/coprocessI", "restarts": "0" },
	{ "exec": "/bin/coprocessJ"}
]
`)
	cfg, _ := NewConfigs(testCfg, nil)
	expectMsg := "expected restarts=%v got %v"

	assert.Equal(t, cfg[0].restartLimit, -1, expectMsg)
	assert.Equal(t, cfg[1].restartLimit, 0, expectMsg)
	assert.Equal(t, cfg[2].restartLimit, 1, expectMsg)
	assert.Equal(t, cfg[3].restartLimit, 1, expectMsg)
	assert.Equal(t, cfg[4].restartLimit, 0, expectMsg)
	assert.Equal(t, cfg[5].restartLimit, 0, expectMsg)
	assert.Equal(t, cfg[6].restartLimit, 0, expectMsg)
}

func TestJobConfigPreStart(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	cfg, err := NewConfigs(testCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg[0].Name, "serviceA", "expected '%v' for serviceA.Name got '%v'")
	assert.Equal(t, cfg[0].stoppingWaitEvent, events.NonEvent,
		"expected '%v' stopping event for serviceA got '%v'")
	assert.Equal(t, cfg[0].whenEvent, events.Event{events.ExitSuccess, "preStart"},
		"expected '%v' for serviceA.whenEvent got '%v'")
	assert.Equal(t, cfg[1].Name, "preStart", "expected '%v' for preStart.Name got '%v'")
	assert.Equal(t, cfg[1].exec.Exec, "/bin/to/preStart.sh",
		"expected '%v' for preStart.exec.Exec got '%v")
}

func TestJobConfigPreStop(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	cfg, err := NewConfigs(testCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg[0].Name, "serviceA", "expected '%v' for serviceA.Name got '%v'")
	assert.Equal(t, cfg[0].stoppingWaitEvent, events.Event{events.Stopped, "preStop"},
		"expected no stopping event for serviceA got '%v'")
	assert.Equal(t, cfg[1].Name, "preStop", "expected '%v' for preStop.Name got '%v'")
	assert.Equal(t, cfg[1].exec.Exec, "/bin/to/preStop.sh",
		"expected '%v' for preStop.exec.Exec got '%v")
	assert.Equal(t, cfg[1].whenEvent, events.Event{events.Stopping, "serviceA"},
		"expected '%v' for preStop.whenEvent got '%v")
}

func TestJobConfigPostStop(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	cfg, err := NewConfigs(testCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, cfg[0].Name, "serviceA", "expected '%v' for serviceA.Name got '%v'")
	assert.Equal(t, cfg[0].stoppingWaitEvent, events.NonEvent,
		"expected no stopping event for serviceA got '%v'")
	assert.Equal(t, cfg[1].Name, "postStop", "expected '%v' for postStop.Name got '%v'")
	assert.Equal(t, cfg[1].exec.Exec, "/bin/to/postStop.sh",
		"expected '%v' for postStop.exec.Exec got '%v")
	assert.Equal(t, cfg[1].whenEvent, events.Event{events.Stopped, "serviceA"},
		"expected '%v' for postStop.whenEvent got '%v")
}
