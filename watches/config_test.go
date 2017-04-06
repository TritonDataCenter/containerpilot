package watches

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
)

func TestWatchesParse(t *testing.T) {
	data, _ := ioutil.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	watches, err := NewConfigs(testCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, watches[0].serviceName, "upstreamA",
		"expected %v for serviceName got %v")
	assert.Equal(t, watches[0].Name, "watch.upstreamA",
		"expected %v for Name got %v")
	assert.Equal(t, watches[0].Poll, 11,
		"expected %v for Poll got %v")

	assert.Equal(t, watches[1].serviceName, "upstreamB",
		"expected %v for serviceName got %v")
	assert.Equal(t, watches[1].Name, "watch.upstreamB",
		"expected %v for Name got %v")
	assert.Equal(t, watches[1].Poll, 79,
		"expected %v for Poll got %v")
}

func TestWatchesConfigError(t *testing.T) {

	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": ""}]`), nil)
	assert.Error(t, err, "`name` must not be blank")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName"}]`), nil)
	assert.Error(t, err, "`poll` must be > 0 in watch myName")
}
