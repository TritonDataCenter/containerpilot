package checks

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
)

func TestCheckParse(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	checks, err := NewConfigs(testCfg)
	if err != nil {
		t.Fatalf("could not parse config JSON: %s", err)
	}

	check0 := checks[0]
	assert.Equal(t, check0.Name, "check.checkA",
		"expected '%v' for check.Name got '%v'")
	assert.Equal(t, check0.Job, "serviceA",
		"expected '%v' for check.Name got '%v'")
	assert.Equal(t, check0.exec.Exec, "/bin/checkA.sh",
		"expected '%v' for exec.Exec got '%v'")
	assert.Equal(t, check0.exec.Args, []string{"A1", "A2"},
		"expected '%v' for exec.Args got '%v'")
	assert.Equal(t, check0.pollInterval, time.Duration(30*time.Second),
		"expected '%v' for check.pollInterval got '%v'")
	assert.Equal(t, check0.timeout, time.Duration(1*time.Millisecond),
		"expected '%v' for check.millisecond got '%v'")

	check1 := checks[1]
	assert.Equal(t, check1.Name, "check.serviceB",
		"expected '%v' for check.Name got '%v'")
	assert.Equal(t, check1.Job, "serviceB",
		"expected '%v' for check.Name got '%v'")
	assert.Equal(t, check1.exec.Exec, "/bin/checkB.sh",
		"expected %v for exec.Exec got %v")
	assert.Equal(t, check1.exec.Args, []string{"B1", "B2"},
		"expected %v for exec.Args got %v")
	assert.Equal(t, check1.pollInterval, time.Duration(30*time.Second),
		"expected '%v' for check.pollInterval got '%v'")
	assert.Equal(t, check1.timeout, time.Duration(30*time.Second),
		"expected '%v' for check.timeout got '%v'")
}

func TestHealthChecksConfigError(t *testing.T) {
	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": "", "exec": "/bin/true"}]`))
	assert.Error(t, err, "`name` must not be blank")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "exec": "/bin/true"}]`))
	assert.Error(t, err, "`poll` must be > 0 in health check check.myName")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "exec": "", "poll": 1}]`))
	assert.Error(t, err,
		"could not parse `exec` in health check check.myName: received zero-length argument")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName", "poll": 1,
		"exec": "/bin/true", "timeout": "xx"}]`))
	assert.Error(t, err,
		"could not parse `timeout` in health check check.myName: time: invalid duration xx")
}
