package jobs

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/joyent/containerpilot/events"
	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/mocks"
)

// ---------------------------------------------------------------------
// Happy path tests

func TestJobConfigServiceWithPreStart(t *testing.T) {
	jobs := loadTestConfig(t)
	assert := assert.New(t)

	// job0 is the main application
	job0 := jobs[0]
	assert.Equal(job0.Name, "serviceA", "config for job0.Name")
	assert.Equal(job0.Exec, "/bin/serviceA.sh", "config for job0.Exec")
	assert.Equal(job0.exec.Exec, "/bin/serviceA.sh",
		"config for job.0.Exec.exec")
	assert.Equal(len(job0.exec.Args), 0, "config for job0.exec.Args")
	assert.Equal(job0.Port, 8080, "config for job0.Port")
	assert.Equal(job0.Tags, []string{"tag1", "tag2"},
		"config for job0.Tags")
	assert.Equal(job0.restartLimit, 0, "config for job0.restartLimit")
	assert.Equal(job0.whenEvent, events.Event{events.ExitSuccess, "preStart"},
		"config for serviceA.whenEvent")
	assert.Equal(job0.healthCheckExec.Exec, "/bin/healthCheckA.sh",
		"config for job0.healthCheckExec.Exec")
	assert.Equal(job0.healthCheckExec.Args, []string{"A1", "A2"},
		"config for job0.healthCheckExec.Args")
	assert.Nil(job0.Restarts, "config for job0.Restarts")

	// job1 is the preStart
	job1 := jobs[1]
	assert.Equal(job1.Name, "preStart", "config for preStart.Name")
	assert.Equal(job1.exec.Exec, "/bin/to/preStart.sh",
		"config for preStart.exec.Exec")
	assert.Equal(job1.whenEvent, events.GlobalStartup,
		"config for job1.whenEvent")
	assert.Equal(job1.Port, 0, "config for job1.Port")
	assert.Equal(job1.restartLimit, 0, "config for job1.restartLimit")
	assert.Nil(job1.Restarts, "config for job1.Restarts")
	assert.Nil(job1.serviceDefinition, "config for job1.serviceDefinition")
}

func TestJobConfigServiceWithArrayExec(t *testing.T) {
	jobs := loadTestConfig(t)
	assert := assert.New(t)

	job0 := jobs[0]
	assert.Equal(job0.Name, "serviceB", "config for job0.Name")
	assert.Equal(job0.Port, 5000, "config for job0.Port")
	assert.Equal(len(job0.Tags), 0, "# of tags in job0.Tags")
	assert.Equal(job0.Exec, []interface{}{"/bin/serviceB", "B"},
		"config for job0.Exec")
	assert.Equal(job0.healthCheckExec.Exec, "/bin/healthCheckB.sh",
		"config for job0.healthCheckExec.Exec")
	assert.Equal(job0.healthCheckExec.Args, []string{"B1", "B2"},
		"config for job0.healthCheckExec.Args")
	assert.Nil(job0.Restarts, "config for job0.Restarts")
}

func TestJobConfigServiceWithStopping(t *testing.T) {
	jobs := loadTestConfig(t)
	assert := assert.New(t)

	// job0 is the main application
	job0 := jobs[0]
	assert.Equal(job0.Name, "serviceA", "config for job0.Name")
	assert.Equal(job0.stoppingWaitEvent, events.Event{events.Stopped, "preStop"},
		"expected no stopping event for serviceA")

	// job1 is its preStart
	job1 := jobs[1]
	assert.Equal(job1.Name, "preStart", "config for job1.Name")

	// job2 is its preStop
	job2 := jobs[2]
	assert.Equal(job2.Name, "preStop", "config for job2.Name")
	assert.Equal(job2.exec.Exec, "/bin/to/preStop.sh",
		"config for preStop.exec.Exec")
	assert.Equal(job2.whenEvent, events.Event{events.Stopping, "serviceA"},
		"config for preStop.whenEvent")

	// job3 is its post-stop
	job3 := jobs[3]
	assert.Equal(job3.Name, "postStop", "config for job3.Name")
	assert.Equal(job3.exec.Exec, "/bin/to/postStop.sh",
		"config for postStop.exec.Exec")
	assert.Equal(job3.whenEvent, events.Event{events.Stopped, "serviceA"},
		"config for postStop.whenEvent")
}

func TestJobConfigServiceNonAdvertised(t *testing.T) {
	job := loadTestConfig(t)[0]
	assert := assert.New(t)
	assert.Equal(job.Name, "coprocessC", "config for job.Name")
	assert.Equal(job.Port, 0, "config for job.Port")
	assert.Equal(job.whenEvent, events.GlobalStartup,
		"config for job.whenEvent")
	assert.Equal(job.Restarts, "unlimited", "config for job.Restarts")
	assert.Equal(job.restartLimit, unlimited, "config for job.restartLimit")
}

func TestJobConfigPeriodicTask(t *testing.T) {
	job := loadTestConfig(t)[0]
	assert := assert.New(t)
	assert.Equal(job.Name, "taskD", "config for job.Name")
	assert.Equal(job.Port, 0, "config for job.Port")
	assert.Equal(job.When.Frequency, "1s", "config for job.When")
	assert.Nil(job.Restarts, "config for job.Restarts") // this the parsed value only
	assert.Equal(job.restartLimit, unlimited, "config for job.restartLimit")
}

func TestJobConfigConsulExtras(t *testing.T) {
	job := loadTestConfig(t)[0]
	assert := assert.New(t)
	assert.Equal(job.Name, "serviceA", "config for job.Name")
	assert.Equal(job.Port, 8080, "config for job.Port")
	assert.Equal(job.ConsulExtras.DeregisterCriticalServiceAfter,
		"10m", "config for job.ConsulExtras.DeregisterCriticalServiceAfter")
	assert.True(job.ConsulExtras.EnableTagOverride,
		"config for job.ConsulExtras.EnableTagOverride")
	assert.Nil(job.Restarts, "config for job.Restarts") // this the parsed value only
	assert.Equal(job.restartLimit, 0, "config.for job.restartLimit")
}

func TestJobConfigSmokeTest(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	assert := assert.New(t)

	jobs, err := NewConfigs(testCfg, noop)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(jobs) != 7 {
		t.Fatalf("expected 7 jobs ", jobs)
	}
	job0 := jobs[0]

	assert.Equal(job0.Name, "serviceA", "config for job0.Name")
	assert.Equal(job0.Exec, "/bin/serviceA", "config for job0.Exec")

	assert.Equal(job0.Port, 8080, "config for job0.Port")
	assert.Equal(job0.Tags, []string{"tag1", "tag2"}, "config for job0.Tags")
	assert.Nil(job0.Restarts, "config for job0.Restarts")

	job1 := jobs[1]
	assert.Equal(job1.Name, "serviceB", "config for job1.Name")
	assert.Equal(job1.Port, 5000, "config for job1.Port")
	assert.Equal(len(job1.Tags), 0, "config for len(job1.Tags)")
	assert.Equal(job1.Exec, []interface{}{"/bin/serviceB", "B"}, "config for job1.Exec")
	assert.Nil(job1.Restarts, "config for job1.Restarts")

	job2 := jobs[2]
	assert.Equal(job2.Name, "coprocessC", "config for job2.Name")
	assert.Equal(job2.Port, 0, "config for job2.Port")
	assert.Equal(job2.When, &WhenConfig{}, "config for job2.When")
	assert.Equal(job2.Restarts, "unlimited", "config for job2.Restarts")

	job3 := jobs[3]
	assert.Equal(job3.Name, "taskD", "config for job3.Name")
	assert.Equal(job3.Port, 0, "config for job3.Port")
	assert.Equal(job3.When.Frequency, "1s", "config for job3.When")
	assert.Nil(job3.Restarts, "config for job3.Restarts")

	job4 := jobs[4]
	assert.Equal(job4.Name, "preStart", "config for job4.Name")
	assert.Equal(job4.Port, 0, "config for job4.Port")
	assert.Equal(job4.When, &WhenConfig{}, "config for job4.When")
	assert.Nil(job4.Restarts, "config for job4.Restarts")

	job5 := jobs[5]
	assert.Equal(job5.Name, "preStop", "config for job5.Name")
	assert.Equal(job5.Port, 0, "config for job5.Port")
	assert.Equal(job5.When, &WhenConfig{Source: "serviceA", Once: "stopping"},
		"config for job5.When")
	assert.Nil(job5.Restarts, "config for job5.Restarts")

	job6 := jobs[6]
	assert.Equal(job6.Name, "postStop", "config for job6.Name")
	assert.Equal(job6.Port, 0, "config for job6.Port")
	assert.Equal(job6.When, &WhenConfig{Source: "serviceA", Once: "stopped"},
		"config for job6.When")
	assert.Nil(job6.Restarts, "config for job6.Restarts")
}

// ---------------------------------------------------------------------
// Error condition tests

func TestJobConfigValidateName(t *testing.T) {
	assert := assert.New(t)

	cfgA := `[{name: "", port: 80, health: {exec: "myhealth", interval: 1, ttl: 3}}]`
	_, err := NewConfigs(tests.DecodeRawToSlice(cfgA), noop)
	assert.Error(err, "'name' must not be blank")

	cfgB := `[{name: "", exec: "myexec", port: 80, health: {exec: "myhealth", interval: 1, ttl: 3}}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgB), noop)
	assert.Error(err, "'name' must not be blank")

	cfgC := `[{name: "", exec: "myexec"}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgC), nil)
	assert.Error(err, "'name' must not be blank")

	// invalid name is permitted if there's no 'port' config
	cfgD := `[{name: "myjob_invalid_name", exec: "myexec"}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgD), noop)
	if err != nil {
		t.Fatal(err)
	}
}

func TestJobConfigValidateDiscovery(t *testing.T) {
	assert := assert.New(t)

	cfgA := `[{name: "myName", port: 80, interfaces: ["inet", "lo0"]}]`
	_, err := NewConfigs(tests.DecodeRawToSlice(cfgA), noop)
	assert.Error(err, "job[myName].health must be set if 'port' is set")

	cfgB := `[{name: "myName", port: 80, interfaces: ["inet", "lo0"], health: {interval: 1}}]`
	_, err = NewConfigs(tests.DecodeRawToSlice(cfgB), noop)
	assert.Error(err, "job[myName].health.ttl must be > 0")

	// no health check shouldn't return an error
	cfgD := `[{name: "myName", port: 80, interfaces: ["inet", "lo0"], health: {interval: 1, ttl: 1}}]`
	if _, err = NewConfigs(tests.DecodeRawToSlice(cfgD), noop); err != nil {
		t.Fatalf("expected no error", err)
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
	assert.Equal(t, job[0].restartLimit, unlimited,
		"expected job[0].restartLimit to be 'unlimited'")
}

func TestJobConfigValidateExec(t *testing.T) {
	assert := assert.New(t)

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
	assert.Equal(cfg[0].exec.Exec, "/bin/serviceA",
		"config for serviceA.exec.Exec")
	assert.Equal(cfg[0].exec.Args, []string{"A1", "A2"},
		"config for serviceA.exec.Args")
	assert.Equal(cfg[0].execTimeout, time.Duration(time.Millisecond),
		"config for serviceA.execTimeout")

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
	assert.Equal(cfg[0].exec.Exec, "/bin/serviceB",
		"config for serviceB.exec.Exec")
	assert.Equal(cfg[0].exec.Args, []string{"B1", "B2"},
		"config for serviceB.exec.Args")
	assert.Equal(cfg[0].execTimeout, time.Duration(time.Millisecond),
		"config for serviceB.execTimeout")

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

	expectErr := func(test, name, val, msg string) {
		errMsg := fmt.Sprintf(`job[%s].restarts field '%s' invalid: %s`, name, val, msg)
		testCfg := tests.DecodeRawToSlice(test)
		_, err := NewConfigs(testCfg, nil)
		assert.Equal(t, err.Error(), errMsg)
	}
	expectErr(
		`[{name: "A", exec: "/bin/coprocessA", restarts: "invalid"}]`,
		"A", "invalid", `accepts positive integers, "unlimited", or "never"`)
	expectErr(
		`[{name: "B", exec: "/bin/coprocessB", restarts: "-1"}]`,
		"B", "-1", `accepts positive integers, "unlimited", or "never"`)
	expectErr(
		`[{name: "C", exec: "/bin/coprocessC", restarts: -1 }]`,
		"C", "-1", `number must be positive integer`)
	expectErr(
		`[{name: "D", exec: "/bin/coprocessD", restarts: "unlimited",
         when: { each: "healthy", source: "other"} }]`,
		"D", "unlimited",
		`may not be used when 'job.when.each' is set because it may result in infinite processes`)

	testCfg := tests.DecodeRawToSlice(`[
	{ name: "D", exec: "/bin/coprocessD", "restarts": "unlimited" },
	{ name: "E", exec: "/bin/coprocessE", "restarts": "never" },
	{ name: "F", exec: "/bin/coprocessF", "restarts": 1 },
	{ name: "G", exec: "/bin/coprocessG", "restarts": "1" },
	{ name: "H", exec: "/bin/coprocessH", "restarts": 0 },
	{ name: "I", exec: "/bin/coprocessI", "restarts": "0" },
	{ name: "J", exec: "/bin/coprocessJ"}]`)

	cfg, _ := NewConfigs(testCfg, nil)
	expectMsg := "expected restartLimit"

	assert := assert.New(t)
	assert.Equal(cfg[0].restartLimit, -1, expectMsg)
	assert.Equal(cfg[1].restartLimit, 0, expectMsg)
	assert.Equal(cfg[2].restartLimit, 1, expectMsg)
	assert.Equal(cfg[3].restartLimit, 1, expectMsg)
	assert.Equal(cfg[4].restartLimit, 0, expectMsg)
	assert.Equal(cfg[5].restartLimit, 0, expectMsg)
	assert.Equal(cfg[6].restartLimit, 0, expectMsg)
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
