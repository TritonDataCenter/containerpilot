package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
This is mostly a giant suite of smoke tests, as the detailed tests of validation
are all in the individual package tests. Here we'll make sure all components
come together as we expect and also check things like env var interpolation.
*/

// jobs.Config
func TestValidConfigJobs(t *testing.T) {

	assert := assert.New(t)
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig("./testdata/test.json5")
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(cfg.Jobs) != 10 {
		t.Fatalf("expected 8 jobs but got %v", cfg.Jobs)
	}
	job0 := cfg.Jobs[0]
	assert.Equal(job0.Name, "serviceA", "config for job0.Name")
	assert.Equal(job0.Port, 8080, "config for job0.Port")
	assert.Equal(job0.Exec, "/bin/serviceA", "config for job0.Exec")
	assert.Equal(job0.Tags, []string{"tag1", "tag2"}, "config for job0.Tags")
	assert.Equal(job0.Meta, map[string]string{"keyA": "A"}, "config for job0.Meta")
	assert.Equal(job0.Restarts, nil, "config for job1.Restarts")

	job1 := cfg.Jobs[1]
	assert.Equal(job1.Name, "serviceB", "config for job1.Name")
	assert.Equal(job1.Port, 5000, "config for job1.Port")
	assert.Equal(len(job1.Tags), 0, "config for len(job1.Tags)")
	assert.Equal(job1.Exec, []interface{}{"/bin/serviceB", "B"}, "config for job1.Exec")
	assert.Equal(job1.Restarts, nil, "config for job1.Restarts")

	job2 := cfg.Jobs[2]
	assert.Equal(job2.Name, "coprocessC", "config for job2.Name")
	assert.Equal(job2.Port, 0, "config for job2.Port")
	assert.Equal(job2.When.Frequency, "", "config for job2.When.Frequency")
	assert.Equal(job2.Restarts, "unlimited", "config for job2.Restarts")

	job3 := cfg.Jobs[3]
	assert.Equal(job3.Name, "periodicTaskD", "config for job3.Name")
	assert.Equal(job3.Port, 0, "config for job3.Port")
	assert.Equal(job3.When.Frequency, "1s", "config for job3.When.Frequency")
	assert.Equal(job3.Restarts, nil, "config for job3.Restarts")

	job4 := cfg.Jobs[4]
	assert.Equal(job4.Name, "preStart", "config for job4.Name")
	assert.Equal(job4.Port, 0, "config for job4.Port")
	assert.Equal(job4.When.Frequency, "", "config for job4.When.Frequency")
	assert.Equal(job4.Restarts, nil, "config for job4.Restarts")

	job5 := cfg.Jobs[5]
	assert.Equal(job5.Name, "preStop", "config for job5.Name")
	assert.Equal(job5.Port, 0, "config for job5.Port")
	assert.Equal(job5.When.Frequency, "", "config for job5.When.Frequency")
	assert.Equal(job5.Restarts, nil, "config for job5.Restarts")

	job6 := cfg.Jobs[6]
	assert.Equal(job6.Name, "postStop", "config for job6.Name")
	assert.Equal(job6.Port, 0, "config for job6.Port")
	assert.Equal(job6.When.Frequency, "", "config for job6.When.Frequency")
	assert.Equal(job6.Restarts, nil, "config for job6.Restarts")

	job7 := cfg.Jobs[7]
	assert.Equal(job7.Name, "onChange-upstreamA", "config for job7.Name")
	assert.Equal(job7.Port, 0, "config for job7.Port")
	assert.Equal(job7.When.Frequency, "", "config for job7.When.Frequency")
	assert.Equal(job7.Restarts, nil, "config for job7.Restarts")

	job8 := cfg.Jobs[8]
	assert.Equal(job8.Name, "onChange-upstreamB", "config for job8.Name")
	assert.Equal(job8.Port, 0, "config for job8.Port")
	assert.Equal(job8.When.Frequency, "", "config for job8.When.Frequency")
	assert.Equal(job8.Restarts, nil, "config for job8.Restarts")

	job9 := cfg.Jobs[9]
	assert.Equal(job9.Name, "containerpilot", "config for job9.Name")
	assert.Equal(job9.Port, 9000, "config for job9.Port")
}

// telemetry.Config
func TestValidConfigTelemetry(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig("./testdata/test.json5")
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	telem := cfg.Telemetry
	metric0 := telem.MetricConfigs[0]
	assert.Equal(t, telem.Port, 9000, "config for telem.Port but got '%v")
	assert.Equal(t, telem.Tags, []string{"dev"}, "config for telem.Tags but got '%v")
	assert.Equal(t, metric0.Name, "zed", "config for metric0.Name but got '%v")
}

// watches.Config
func TestValidConfigWatches(t *testing.T) {
	os.Setenv("TEST", "HELLO")
	cfg, err := LoadConfig("./testdata/test.json5")
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}

	if len(cfg.Watches) != 2 {
		t.Fatalf("expected 2 watches but got %v", cfg.Watches)
	}
	watch0 := cfg.Watches[0]
	watch1 := cfg.Watches[1]
	assert.Equal(t, watch0.Name, "watch.upstreamA", "config for Name")
	assert.Equal(t, watch0.Poll, 11, "config for Poll")
	assert.Equal(t, watch0.Tag, "dev", "config for Tag")
	assert.Equal(t, watch1.Name, "watch.upstreamB", "config for Name")
	assert.Equal(t, watch1.Poll, 79, "config for Poll")
	assert.Equal(t, watch1.Tag, "", "config for Tag")
}

// control.Config
func TestValidConfigControl(t *testing.T) {
	cfg, err := LoadConfig("./testdata/test.json5")
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}
	assert.Equal(t, cfg.Control.SocketPath,
		"/var/run/containerpilot.socket",
		"config for control.socket")
}

func TestCustomConfigControl(t *testing.T) {
	var testJSONWithSocket = `{
	"control": {"socket": "/var/run/cp3-test.sock"},
	"consul": "consul:8500"}`

	cfg, err := newConfig([]byte(testJSONWithSocket))
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}
	assert.Equal(t, cfg.Control.SocketPath,
		"/var/run/cp3-test.sock",
		"config for control.socket")
}

func TestInvalidRenderConfigFileMissing(t *testing.T) {
	err := RenderConfig("/xxxx", "-")
	assert.Error(t, err,
		"could not read config file: open /xxxx: no such file or directory")
}

func TestInvalidRenderConfigOutputMissing(t *testing.T) {
	err := RenderConfig("./testdata/test.json5", "./xxxx/xxxx")
	assert.Error(t, err,
		"could not write config file: open ./xxxx/xxxx: no such file or directory")
}

func TestRenderConfigFileStdout(t *testing.T) {

	// Render to file
	defer os.Remove("testJSON.json")
	if err := RenderConfig("./testdata/test.json5", "testJSON.json"); err != nil {
		t.Fatalf("expected no error from renderConfigTemplate but got: %v", err)
	}
	if exists, err := fileExists("testJSON.json"); !exists || err != nil {
		t.Errorf("expected file testJSON.json to exist.")
	}

	// Render to stdout
	fname := filepath.Join(os.TempDir(), "stdout")
	temp, _ := os.Create(fname)
	old := os.Stdout
	os.Stdout = temp
	if err := RenderConfig("./testdata/test.json5", "-"); err != nil {
		t.Fatalf("expected no error from renderConfigTemplate but got: %v", err)
	}
	temp.Close()
	os.Stdout = old

	renderedOut, _ := os.ReadFile(fname)
	renderedFile, _ := os.ReadFile("testJSON.json")
	if string(renderedOut) != string(renderedFile) {
		t.Fatalf("expected the rendered file and stdout to be identical")
	}
}

func TestRenderedConfigIsParseable(t *testing.T) {

	var testJSON = `{
	"consul": "consul:8500",
	watches: [{"name": "upstreamA{{.TESTRENDERCONFIGISPARSEABLE}}", "interval": 11}]}`

	os.Setenv("TESTRENDERCONFIGISPARSEABLE", "-ok")
	template, _ := renderConfigTemplate([]byte(testJSON))
	config, err := newConfig(template)
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}
	name := config.Watches[0].Name
	if name != "watch.upstreamA-ok" {
		t.Fatalf("expected Watches[0] name to be upstreamA-ok but got %s", name)
	}
}

// ----------------------------------------------------
// test helpers

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
