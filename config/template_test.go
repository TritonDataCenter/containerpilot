package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	// need to import this so that we have a registered backend
	_ "github.com/joyent/containerpilot/discovery/consul"
	"github.com/joyent/containerpilot/tests/assert"
)

func TestParseEnvironment(t *testing.T) {
	parsed := parseEnvironment([]string{})
	assert.Equal(t, parsed, Environment{}, "expected environment '%v' got '%v'")

	parsed = parseEnvironment([]string{
		"VAR1=test",
		"VAR2=test2",
	})
	assert.Equal(t, parsed, Environment{
		"VAR1": "test",
		"VAR2": "test2",
	}, "expected environment '%v' got '%v'")
}

func TestTemplate(t *testing.T) {
	env := parseEnvironment([]string{"NAME=Template", "USER=pilot", "PARTS=a:b:c"})
	// To test env function
	os.Setenv("NAME_1", "Template")
	defer os.Unsetenv("NAME_1")

	testTemplate := func(name string, template string, expected string) {
		tmpl, err := NewTemplate([]byte(template))
		if err != nil {
			t.Fatalf("%s - error parsing template: %s", name, err)
		}
		tmpl.Env = env
		res, err2 := tmpl.Execute()
		if err2 != nil {
			t.Fatalf("%s - error executing template: %s", name, err2)
		}
		strRes := string(res)
		if strRes != expected {
			t.Fatalf("%s - expected %s but got: %s", name, expected, strRes)
		}
	}

	testTemplate("One var", `Hello, {{.NAME}}!`, "Hello, Template!")
	testTemplate("Var undefined", `Hello, {{.NONAME}}!`, "Hello, !")
	testTemplate("Loop double", `{{ loop 2 5 }}`, "[2 3 4]")
	testTemplate("Loop inverse", `{{ loop 10 1 }}`, "[10 9 8 7 6 5 4 3 2]")
	testTemplate("Loop single", `{{ loop 5 }}`, "[0 1 2 3 4]")
	testTemplate("Loop range",
		`{{ range $i := loop 2 5 -}}i={{$i}},{{ end }}`, "i=2,i=3,i=4,")
	testTemplate("ENV", `Hello, {{ env (printf "NA%s" "ME_1") }}!`, "Hello, Template!")
	testTemplate("Default", `Hello, {{.NONAME | default "World" }}!`, "Hello, World!")
	testTemplate("Default", `Hello, {{.NONAME | default 100 }}!`, "Hello, 100!")
	testTemplate("Default", `Hello, {{.NONAME | default 10.1 }}!`, "Hello, 10.1!")
	testTemplate("Split and Join",
		`Hello, {{.PARTS | split ":" | join "." }}!`, "Hello, a.b.c!")
	testTemplate("Replace All",
		`Hello, {{.NAME | replaceAll "e" "_" }}!`, "Hello, T_mplat_!")
	testTemplate("Regex Replace All",
		`Hello, {{.NAME | regexReplaceAll "[epa]+" "_" }}!`, "Hello, T_m_l_t_!")
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

	renderedOut, _ := ioutil.ReadFile(fname)
	renderedFile, _ := ioutil.ReadFile("testJSON.json")
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
