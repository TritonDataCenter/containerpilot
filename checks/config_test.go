package checks

import (
	"fmt"
	"io/ioutil"
	"testing"

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
	assert.Equal(t, checks[0].exec.Exec, "/bin/checkA.sh",
		"expected %v for exec.Exec got %v")
	assert.Equal(t, checks[0].exec.Args, []string{"A1", "A2"},
		"expected %v for exec.Args got %v")

	assert.Equal(t, checks[1].exec.Exec, "/bin/checkB.sh",
		"expected %v for exec.Exec got %v")
	assert.Equal(t, checks[1].exec.Args, []string{"B1", "B2"},
		"expected %v for exec.Args got %v")
}

func TestHealthChecksConfigError(t *testing.T) {
	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": "", "health": "/bin/true"}]`))
	assert.Error(t, err, "`name` must not be blank")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "health": "/bin/true"}]`))
	assert.Error(t, err, "`poll` must be > 0 in health check myName.check")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "health": "", "poll": 1}]`))
	assert.Error(t, err, "could not parse `health` in check myName.check: received zero-length argument")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName", "poll": 1, "ttl": 1,
		"port": 80, "health": "/bin/true", "timeout": "xx"}]`))
	assert.Error(t, err,
		"could not parse `timeout` in check myName.check: time: invalid duration xx")
}
