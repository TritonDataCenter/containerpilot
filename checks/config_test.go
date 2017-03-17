package checks

import (
	"testing"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
)

func TestCheckParse(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[
{
  "name": "serviceA",
  "port": 8080,
  "interfaces": "inet",
  "health": ["/bin/checkA.sh", "A1", "A2"],
  "poll": 30,
  "ttl": 19,
  "timeout": "1ms",
  "tags": ["tag1","tag2"]
},
{
  "name": "serviceB",
  "port": 5000,
  "interfaces": ["ethwe","eth0", "inet"],
  "health": "/bin/checkB.sh B1 B2",
  "poll": 30,
  "ttl": 103
}
]`)
	checks, err := NewConfigs(testCfg)
	if err != nil {
		t.Fatalf("could not parse service JSON: %s", err)
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
	assert.Error(t, err, "`poll` must be > 0 in health check myName")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "health": "", "poll": 1}]`))
	assert.Error(t, err, "could not parse `health` in check myName: received zero-length argument")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName", "poll": 1, "ttl": 1,
		"port": 80, "health": "/bin/true", "timeout": "xx"}]`))
	assert.Error(t, err,
		"could not parse `timeout` in check myName: time: invalid duration xx")
}
