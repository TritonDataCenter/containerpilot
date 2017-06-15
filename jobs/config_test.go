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

// ---------------------------------------------------------------------
// Happy path tests

func TestJobConfigServiceWithPreStart(t *testing.T) {
	jobs := loadTestConfig(t)

	// job0 is the main application
	job0 := jobs[0]
	assert.Equal(t, job0.Name, "serviceA",
		"expected '%v' for job0.Name but got '%v'")
	assert.Equal(t, job0.Exec, "/bin/serviceA.sh",
		"expected '%v' for job0.Exec but got '%v'")
	assert.Equal(t, job0.exec.Exec, "/bin/serviceA.sh",
		"expected '%v' for job0.exec.Exec but got '%v'")
	assert.Equal(t, len(job0.exec.Args), 0,
		"expected '%v' for len(job0.exec.Args) but got '%v'")
	assert.Equal(t, job0.Port, 8080,
		"expected '%v' for job0.Port but got '%v'")
	assert.Equal(t, job0.Tags, []string{"tag1", "tag2"},
		"expected '%v' for job0.Tags but got '%v'")
	assert.Equal(t, job0.restartLimit, 0,
		"expected '%v' for job0.restartLimit but got '%v'")
	assert.Equal(t, job0.whenEvent, events.Event{events.ExitSuccess, "preStart"},
		"expected '%v' for serviceA.whenEvent got '%v'")
	assert.Equal(t, job0.healthCheckExec.Exec, "/bin/healthCheckA.sh",
		"expected %v for job0.healthCheckExec.Exec got %v")
	assert.Equal(t, job0.healthCheckExec.Args, []string{"A1", "A2"},
		"expected %v for job0.healthCheckExec.Args got %v")
	if job0.Restarts != nil {
		t.Fatalf("expected nil for job0.Restarts but got '%v'", job0.Restarts)
	}

	// job1 is the preStart
	job1 := jobs[1]
	assert.Equal(t, job1.Name, "preStart",
		"expected '%v' for preStart.Name got '%v'")
	assert.Equal(t, job1.exec.Exec, "/bin/to/preStart.sh",
		"expected '%v' for preStart.exec.Exec got '%v")
	assert.Equal(t, job1.whenEvent, events.GlobalStartup,
		"expected '%v' for job1.whenEvent got '%v'")
	assert.Equal(t, job1.Port, 0,
		"expected '%v' for job1.Port but got '%v'")
	assert.Equal(t, job1.restartLimit, 0,
		"expected '%v' for job1.restartLimit got '%v'")
	if job1.Restarts != nil {
		t.Fatalf("expected nil for job1.Restarts but got '%v'", job1.Restarts)
	}
	if job1.serviceDefinition != nil {
		t.Fatalf("expected nil for job1.serviceDefinition but got '%v'",
			job1.serviceDefinition)
	}

}

func TestJobConfigServiceWithArrayExec(t *testing.T) {
	jobs := loadTestConfig(t)
	job0 := jobs[0]
	assert.Equal(t, job0.Name, "serviceB",
		"expected '%v' for job0.Name but got '%v'")
	assert.Equal(t, job0.Port, 5000,
		"expected '%v' for job0.Port but got '%v'")
	assert.Equal(t, len(job0.Tags), 0,
		"expected '%v' for len(job0.Tags) but got '%v'")
	assert.Equal(t, job0.Exec, []interface{}{"/bin/serviceB", "B"},
		"expected '%v' for job0.Exec but got '%v'")
	assert.Equal(t, job0.healthCheckExec.Exec, "/bin/healthCheckB.sh",
		"expected %v for exec.Exec got %v")
	assert.Equal(t, job0.healthCheckExec.Args, []string{"B1", "B2"},
		"expected %v for exec.Args got %v")
	if job0.Restarts != nil {
		t.Fatalf("expected nil for job0.Restarts but got '%v'", job0.Restarts)
	}
}

func TestJobConfigServiceWithStopping(t *testing.T) {
	jobs := loadTestConfig(t)

	// job0 is the main application
	job0 := jobs[0]
	assert.Equal(t, job0.Name, "serviceA",
		"expected '%v' for job0.Name but got '%v'")
	assert.Equal(t, job0.Name, "serviceA",
		"expected '%v' for serviceA.Name got '%v'")
	assert.Equal(t, job0.stoppingWaitEvent, events.Event{events.Stopped, "preStop"},
		"expected no stopping event for serviceA got '%v'")

	// job1 is its preStart
	job1 := jobs[1]
	assert.Equal(t, job1.Name, "preStart",
		"expected '%v' for preStart.Name got '%v'")

	// job2 is its preStop
	job2 := jobs[2]
	assert.Equal(t, job2.Name, "preStop",
		"expected '%v' for preStop.Name got '%v'")
	assert.Equal(t, job2.exec.Exec, "/bin/to/preStop.sh",
		"expected '%v' for preStop.exec.Exec got '%v")
	assert.Equal(t, job2.whenEvent, events.Event{events.Stopping, "serviceA"},
		"expected '%v' for preStop.whenEvent got '%v")

	// job3 is its post-stop
	job3 := jobs[3]
	assert.Equal(t, job3.Name, "postStop",
		"expected '%v' for postStop.Name got '%v'")
	assert.Equal(t, job3.exec.Exec, "/bin/to/postStop.sh",
		"expected '%v' for postStop.exec.Exec got '%v")
	assert.Equal(t, job3.whenEvent, events.Event{events.Stopped, "serviceA"},
		"expected '%v' for postStop.whenEvent got '%v")
}

func TestJobConfigServiceNonAdvertised(t *testing.T) {
	job := loadTestConfig(t)[0]
	assert.Equal(t, job.Name, "coprocessC", "expected '%v' for job.Name but got '%v'")
	assert.Equal(t, job.Port, 0, "expected '%v' for job.Port but got '%v'")
	assert.Equal(t, job.whenEvent, events.GlobalStartup,
		"expected '%v' for job.whenEvent but got '%v'")
	assert.Equal(t, job.Restarts, "unlimited", "expected '%v' for job.Restarts but got '%v'")
}

func TestJobConfigPeriodicTask(t *testing.T) {
	job := loadTestConfig(t)[0]
	assert.Equal(t, job.Name, "taskD", "expected '%v' for job.Name but got '%v'")
	assert.Equal(t, job.Port, 0, "expected '%v' for job.Port but got '%v'")
	assert.Equal(t, job.When.Frequency, "1s", "expected '%v' for job.When but got '%v'")
	if job.Restarts != nil {
		t.Fatalf("expected nil for job.Restarts but got '%v'", job.Restarts)
	}
}

func TestJobConfigConsulExtras(t *testing.T) {
	job := loadTestConfig(t)[0]
	assert.Equal(t, job.Name, "serviceA", "expected '%v' for job.Name but got '%v'")
	assert.Equal(t, job.Port, 8080, "expected '%v' for job.Port but got '%v'")
	assert.Equal(t, job.ConsulExtras.DeregisterCriticalServiceAfter,
		"10m", "expected %v got %v")
	assert.Equal(t, job.ConsulExtras.EnableTagOverride,
		true, "expected %v got %v")
	if job.Restarts != nil {
		t.Fatalf("expected nil for job.Restarts but got '%v'", job.Restarts)
	}

}

func TestJobConfigSmokeTest(t *testing.T) {
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
	assert.Equal(t, job0.Exec, "/bin/serviceA", "expected '%v' for job0.Exec but got '%v'")

	assert.Equal(t, job0.Port, 8080, "expected '%v' for job0.Port but got '%v'")
	assert.Equal(t, job0.Tags, []string{"tag1", "tag2"}, "expected '%v' for job0.Tags but got '%v'")
	if job0.Restarts != nil {
		t.Fatalf("expected nil for job0.Restarts but got '%v'", job0.Restarts)
	}

	job1 := jobs[1]
	assert.Equal(t, job1.Name, "serviceB", "expected '%v' for job1.Name but got '%v'")
	assert.Equal(t, job1.Port, 5000, "expected '%v' for job1.Port but got '%v'")
	assert.Equal(t, len(job1.Tags), 0, "expected '%v' for len(job1.Tags) but got '%v'")
	assert.Equal(t, job1.Exec, []interface{}{"/bin/serviceB", "B"}, "expected '%v' for job1.Exec but got '%v'")
	if job1.Restarts != nil {
		t.Fatalf("expected nil for job1.Restarts but got '%v'", job1.Restarts)
	}

	job2 := jobs[2]
	assert.Equal(t, job2.Name, "coprocessC", "expected '%v' for job2.Name but got '%v'")
	assert.Equal(t, job2.Port, 0, "expected '%v' for job2.Port but got '%v'")
	assert.Equal(t, job2.When, &WhenConfig{}, "expected '%v' for job2.When but got '%v'")
	assert.Equal(t, job2.Restarts, "unlimited", "expected '%v' for job2.Restarts but got '%v'")

	job3 := jobs[3]
	assert.Equal(t, job3.Name, "taskD", "expected '%v' for job3.Name but got '%v'")
	assert.Equal(t, job3.Port, 0, "expected '%v' for job3.Port but got '%v'")
	assert.Equal(t, job3.When.Frequency, "1s", "expected '%v' for job3.When but got '%v'")
	if job3.Restarts != nil {
		t.Fatalf("expected nil for job3.Restarts but got '%v'", job3.Restarts)
	}

	job4 := jobs[4]
	assert.Equal(t, job4.Name, "preStart", "expected '%v' for job4.Name but got '%v'")
	assert.Equal(t, job4.Port, 0, "expected '%v' for job4.Port but got '%v'")
	assert.Equal(t, job4.When, &WhenConfig{}, "expected '%v' for job4.When but got '%v'")
	if job4.Restarts != nil {
		t.Fatalf("expected nil for job4.Restarts but got '%v'", job4.Restarts)
	}

	job5 := jobs[5]
	assert.Equal(t, job5.Name, "preStop", "expected '%v' for job5.Name but got '%v'")
	assert.Equal(t, job5.Port, 0, "expected '%v' for job5.Port but got '%v'")
	assert.Equal(t, job5.When, &WhenConfig{Source: "serviceA", Once: "stopping"},
		"expected '%v' for job5.When but got '%v'")
	if job5.Restarts != nil {
		t.Fatalf("expected nil for job5.Restarts but got '%v'", job5.Restarts)
	}

	job6 := jobs[6]
	assert.Equal(t, job6.Name, "postStop", "expected '%v' for job6.Name but got '%v'")
	assert.Equal(t, job6.Port, 0, "expected '%v' for job6.Port but got '%v'")
	assert.Equal(t, job6.When, &WhenConfig{Source: "serviceA", Once: "stopped"},
		"expected '%v' for job6.When but got '%v'")
	if job6.Restarts != nil {
		t.Fatalf("expected nil for job6.Restarts but got '%v'", job6.Restarts)
	}
}

// ---------------------------------------------------------------------
// Error condition tests

func TestJobConfigValidateName(t *testing.T) {

	cfgA := `[{name: "", port: 80, health: {exec: "myhealth", interval: 1, ttl: 3}}]`
	_, err := NewConfigs(tests.DecodeRawToSlice(cfgA), noop)
	assert.Error(t, err, "'name' must not be blank")

	cfgB := `[{name: "", exec: "myexec", port: 80, health: {exec: "myhealth", interval: 1, ttl: 3}}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgB), noop)
	assert.Error(t, err, "'name' must not be blank")

	cfgC := `[{name: "", exec: "myexec"}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgC), nil)
	assert.Error(t, err, "'name' must not be blank")

	// invalid name is permitted if there's no 'port' config
	cfgD := `[{name: "myjob_invalid_name", exec: "myexec"}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgD), noop)
	if err != nil {
		t.Fatal(err)
	}
}

func TestJobConfigValidateDiscovery(t *testing.T) {

	cfgA := `[{name: "myName", port: 80}]`
	_, err := NewConfigs(tests.DecodeRawToSlice(cfgA), noop)
	assert.Error(t, err, "job[myName].health must be set if 'port' is set")

	cfgB := `[{name: "myName", port: 80, health: {interval: 1}}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgB), noop)
	assert.Error(t, err, "job[myName].health.ttl must be > 0")

	// no health check shouldn't return an error
	cfgD := `[{name: "myName", port: 80, health: {interval: 1, ttl: 1}}]`
	if _, err = NewConfigs(tests.DecodeRawToSlice(cfgD), noop); err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
}

func TestErrJobConfigConsulEnableTagOverride(t *testing.T) {
	testCfg, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	_, err := NewConfigs(tests.DecodeRawToSlice(string(testCfg)), noop)
	if err == nil {
		t.Errorf("ConsulExtras should have thrown error about EnableTagOverride being a string.")
	}
}

func TestErrJobConfigConsulDeregisterCriticalServiceAfter(t *testing.T) {
	testCfg, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	_, err := NewConfigs(tests.DecodeRawToSlice(string(testCfg)), noop)
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
	expectErr(
		`[{name: "A", exec: "/bin/taskA", timeout: "1s", when: {interval: "-1s"}}]`,
		"job[A].when.interval '-1s' cannot be less than 1ms")

	expectErr(
		`[{name: "B", exec: "/bin/taskB", timeout: "1s", when: {interval: "1ns"}}]`,
		"job[B].when.interval '1ns' cannot be less than 1ms")

	expectErr(
		`[{name: "C", exec: "/bin/taskC", timeout: "-1ms", when: {interval: "1ms"}}]`,
		"job[C].timeout '-1ms' cannot be less than 1ms")

	expectErr(
		`[{name: "D", exec: "/bin/taskD", timeout: "1ns", when: {interval: "1ms"}}]`,
		"job[D].timeout '1ns' cannot be less than 1ms")

	expectErr(
		`[{name: "E", exec: "/bin/taskE", timeout: "1ns", when: {interval: "xx"}}]`,
		"unable to parse job[E].when.interval 'xx': time: invalid duration xx")

	testCfg := tests.DecodeRawToSlice(
		`[{name: "F", exec: "/bin/taskF", when: {interval: "1ms"}}]`)
	job, _ := NewConfigs(testCfg, nil)
	assert.Equal(t, job[0].execTimeout, job[0].freqInterval,
		"expected execTimeout '%v' to equal interval '%v'")
}

func TestJobConfigValidateExec(t *testing.T) {

	testCfg := tests.DecodeRawToSlice(`[
	{
		name: "serviceA",
		exec: ["/bin/serviceA", "A1", "A2"],
		timeout: "1ms"
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
		name: "serviceB",
		exec: "/bin/serviceB B1 B2",
		timeout: "1ms"
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
		name: "serviceC",
		exec: "/bin/serviceC C1 C2",
		timeout: "xx"
	}]`)
	_, err = NewConfigs(testCfg, noop)
	expected := "unable to parse job[serviceC].timeout 'xx': time: invalid duration xx"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected '%s', got '%v'", expected, err)
	}

	testCfg = tests.DecodeRawToSlice(`[
	{
		name: "serviceD",
		exec: ""
	}]`)
	_, err = NewConfigs(testCfg, noop)
	expected = "unable to create job[serviceD].exec: received zero-length argument"
	if err == nil || err.Error() != expected {
		t.Fatalf("expected '%s', got '%v'", expected, err)
	}

}

func TestJobConfigValidateRestarts(t *testing.T) {

	expectErr := func(test, name, val string) {
		errMsg := fmt.Sprintf(`job[%s].restarts field '%s' invalid: accepts positive integers, "unlimited", or "never"`, name, val)
		testCfg := tests.DecodeRawToSlice(test)
		_, err := NewConfigs(testCfg, nil)
		assert.Error(t, err, errMsg)
	}
	expectErr(
		`[{name: "A", exec: "/bin/coprocessA", "restarts": "invalid"}]`,
		"A", "invalid")
	expectErr(
		`[{name: "B", exec: "/bin/coprocessB", "restarts": "-1"}]`,
		"B", "-1")
	expectErr(
		`[{name: "C", exec: "/bin/coprocessC", "restarts": -1 }]`,
		"C", "-1")

	testCfg := tests.DecodeRawToSlice(`[
	{ name: "D", exec: "/bin/coprocessD", "restarts": "unlimited" },
	{ name: "E", exec: "/bin/coprocessE", "restarts": "never" },
	{ name: "F", exec: "/bin/coprocessF", "restarts": 1 },
	{ name: "G", exec: "/bin/coprocessG", "restarts": "1" },
	{ name: "H", exec: "/bin/coprocessH", "restarts": 0 },
	{ name: "I", exec: "/bin/coprocessI", "restarts": "0" },
	{ name: "J", exec: "/bin/coprocessJ"}]`)

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

func TestHealthChecksConfigError(t *testing.T) {

	expectErr := func(test, errMsg string) {
		testCfg := tests.DecodeRawToSlice(test)
		_, err := NewConfigs(testCfg, nil)
		assert.Error(t, err, errMsg)
	}
	expectErr(
		`[{name: "myName", health: {exec: "/bin/true"}}]`,
		"job[myName].health.interval must be > 0")
	expectErr(
		`[{name: "myName", health: {exec: "/bin/true", interval: 1}}]`,
		"job[myName].health.ttl must be > 0")
	expectErr(
		`[{name: "myName", health: {exec: "", interval: 1, ttl: 5}}]`,
		"unable to create job[myName].health.exec: received zero-length argument")
	expectErr(
		`[{name: "myName", health: {exec: "/bin/true", interval: 1, ttl: 5, timeout: "xx"}}]`,
		"could not parse job[myName].health.timeout 'xx': time: invalid duration xx")
}

// ---------------------------------------------------------------------
// helpers

var noop = &mocks.NoopDiscoveryBackend{}

func loadTestConfig(t *testing.T) []*Config {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))

	jobs, err := NewConfigs(testCfg, noop)
	if err != nil {
		t.Fatalf("unexpected error in '%s' for LoadConfig: %v", t.Name(), err)
	}
	return jobs
}
