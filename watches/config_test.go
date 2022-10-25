package watches

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tritondatacenter/containerpilot/tests"
)

func TestWatchesParse(t *testing.T) {
	data, _ := os.ReadFile(fmt.Sprintf("./testdata/%s.json5", t.Name()))
	testCfg := tests.DecodeRawToSlice(string(data))
	watches, err := NewConfigs(testCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert := assert.New(t)
	assert.Equal(watches[0].serviceName, "upstreamA", "config for serviceName")
	assert.Equal(watches[0].Name, "watch.upstreamA", "config for Name")
	assert.Equal(watches[0].Poll, 11, "config for Poll")
	assert.Equal(watches[0].Tag, "dev", "config for Tag")
	assert.Equal(watches[0].DC, "", "config for DC")

	assert.Equal(watches[1].serviceName, "upstreamB", "config for serviceName")
	assert.Equal(watches[1].Name, "watch.upstreamB", "config for Name")
	assert.Equal(watches[1].Poll, 79, "config for Poll")
	assert.Equal(watches[1].DC, "us-east-1", "config for DC")
}

func TestWatchesConfigError(t *testing.T) {

	_, err := NewConfigs(tests.DecodeRawToSlice(`[{"name": ""}]`), nil)
	assert.Error(t, err, "'name' must not be blank")

	_, err = NewConfigs(tests.DecodeRawToSlice(
		`[{"name": "myName"}]`), nil)
	assert.Error(t, err, "watch[myName].interval must be > 0")
}
