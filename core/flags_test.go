package core

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidConfigNoConfigFlag(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "/testdata/test.sh", "invalid1", "--debug"}
	_, p := GetArgs()
	if _, err := NewApp(p.ConfigPath); err != nil && err.Error() != "-config flag is required" {
		t.Errorf("expected error but got %s", err)
	}
}

func TestInvalidConfigParseNoDiscovery(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	f1 := testCfgToTempFile(t, "{}")
	defer os.Remove(f1.Name())
	os.Args = []string{"this", "-config", f1.Name()}
	_, p := GetArgs()
	_, err := NewApp(p.ConfigPath)
	assert.Error(t, err, "no discovery backend defined")
}

func TestInvalidConfigMissingFile(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "/xxxx"}
	_, p := GetArgs()
	_, err := NewApp(p.ConfigPath)
	assert.Error(t, err,
		"could not read config file: open /xxxx: no such file or directory")
}

func TestInvalidConfigParseNotJson(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	f1 := testCfgToTempFile(t, "<>")
	defer os.Remove(f1.Name())
	os.Args = []string{"this", "-config", f1.Name()}
	_, p := GetArgs()
	_, err := NewApp(p.ConfigPath)
	assert.Error(t, fmt.Errorf("%s", err.Error()[:29]),
		"parse error at line:col [1:1]")
}

func TestInvalidConfigParseTemplateError(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	// this config is missing quotes around the template
	f1 := testCfgToTempFile(t, `{"test": {{ .NO_SUCH_KEY }}, "test2": "hello"}`)
	defer os.Remove(f1.Name())
	os.Args = []string{"this", "-config", f1.Name()}
	_, p := GetArgs()
	_, err := NewApp(p.ConfigPath)
	assert.Error(t, fmt.Errorf("%s", err.Error()[:30]),
		"parse error at line:col [1:10]")
}

func TestControlServerCreation(t *testing.T) {
	f1 := testCfgToTempFile(t, `{"consul": "consul:8500"}`)
	defer os.Remove(f1.Name())
	app, err := NewApp(f1.Name())
	if err != nil {
		t.Fatalf("got error while initializing config: %v", err)
	}
	if app.ControlServer == nil {
		t.Error("expected control server to not be nil")
	}
}

func TestPidEnvVar(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "{}", "/testdata/test.sh"}
	_, p := GetArgs()
	NewApp(p.ConfigPath)
	if pid := os.Getenv("CONTAINERPILOT_PID"); pid == "" {
		t.Errorf("expected CONTAINERPILOT_PID to be set even on error")
	}
}

func TestSetEqual(t *testing.T) {
	defer argTestCleanup(argTestSetup())
	os.Args = []string{"this", "-config", "{}", "-putenv", "ENV_VALUE=PART1=PART2"}
	_, p := GetArgs()
	if value, ok := p.Env["ENV_VALUE"]; !ok || value != "PART1=PART2" {
		t.Errorf("expected ENV_VALUE to be set to 'PART1=PART2'")
	}
}

// ----------------------------------------------------
// test helpers

func argTestSetup() []string {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.Usage = nil
	return os.Args
}

func argTestCleanup(oldArgs []string) {
	os.Args = oldArgs
}
