package watches

import (
	"testing"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
)

func TestWatchesParse(t *testing.T) {
	testCfg := tests.DecodeRawToSlice(`[
{
  "name": "upstreamA",
  "poll": 11,
  "onChange": ["/bin/upstreamA.sh", "A1", "A2"],
  "tag": "dev"
},
{
  "name": "upstreamB",
  "poll": 79,
  "onChange": "/bin/upstreamB.sh B1 B2"
}
]`)
	watches, err := NewConfigs(testCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, watches[0].exec.Exec, "/bin/upstreamA.sh",
		"expected %v for exec.Exec got %v")
	assert.Equal(t, watches[0].exec.Args, []string{"A1", "A2"},
		"expected %v for exec.Args got %v")
	assert.Equal(t, watches[1].exec.Exec, "/bin/upstreamB.sh",
		"expected %v for exec.Exec got %v")
	assert.Equal(t, watches[1].exec.Args, []string{"B1", "B2"},
		"expected %v for exec.Args got %v")
}

func TestWatchesConfigError(t *testing.T) {

	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": ""}]`), nil)
	assert.Error(t, err, "`name` must not be blank")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName"}]`), nil)
	assert.Error(t, err, "`onChange` is required in watch myName")

	_, err = NewConfigs(tests.DecodeRawToSlice(`[{"name": "myName", "onChange": "", "poll": 1}]`), nil)
	assert.Error(t, err, "could not parse `onChange` in watch myName: received zero-length argument")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName", "onChange": "true", "poll": 1, "timeout": "xx"}]`), nil)
	assert.Error(t, err,
		"could not parse `timeout` in watch myName: time: invalid duration xx")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName", "onChange": "true", "timeout": ""}]`), nil)
	assert.Error(t, err, "`poll` must be > 0 in watch myName")
}
