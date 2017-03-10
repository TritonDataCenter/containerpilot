package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	// need to import this so that we have a registered backend
	_ "github.com/joyent/containerpilot/discovery/consul"
)

func TestParseEnvironment(t *testing.T) {
	validateParseEnvironment(t, "Empty Environment", []string{}, Environment{})
	validateParseEnvironment(t, "Example Environment", []string{
		"VAR1=test",
		"VAR2=test2",
	}, Environment{
		"VAR1": "test",
		"VAR2": "test2",
	})
}

func TestTemplate(t *testing.T) {
	env := parseEnvironment([]string{"NAME=Template", "USER=pilot", "PARTS=a:b:c"})
	validateTemplate(t, "One var", `Hello, {{.NAME}}!`, env, "Hello, Template!")
	validateTemplate(t, "Var undefined", `Hello, {{.NONAME}}!`, env, "Hello, !")
	validateTemplate(t, "Default", `Hello, {{.NONAME | default "World" }}!`, env, "Hello, World!")
	validateTemplate(t, "Default", `Hello, {{.NONAME | default 100 }}!`, env, "Hello, 100!")
	validateTemplate(t, "Default", `Hello, {{.NONAME | default 10.1 }}!`, env, "Hello, 10.1!")
	validateTemplate(t, "Split and Join", `Hello, {{.PARTS | split ":" | join "." }}!`, env, "Hello, a.b.c!")
	validateTemplate(t, "Replace All", `Hello, {{.NAME | replaceAll "e" "_" }}!`, env, "Hello, T_mplat_!")
	validateTemplate(t, "Regex Replace All", `Hello, {{.NAME | regexReplaceAll "[epa]+" "_" }}!`, env, "Hello, T_m_l_t_!")
}

func TestInvalidRenderConfigFile(t *testing.T) {
	testRenderExpectError(t, "file:///xxxx", "-",
		"could not read config file: open /xxxx: no such file or directory")
}

func TestInvalidRenderFileConfig(t *testing.T) {
	var testJSON = `{"consul": "consul:8500"}`
	testRenderExpectError(t, testJSON, "file:///a/b/c/d/e/f.json",
		"could not write config file: open /a/b/c/d/e/f.json: no such file or directory")
}

func TestRenderConfigFileStdout(t *testing.T) {

	var testJSON = `{
	"consul": "consul:8500",
	"backends": [{
					"name": "upstreamA",
					"poll": 11,
					"onChange": "/bin/to/onChangeEvent/for/upstream/A.sh"}]}`

	// Render to file
	defer os.Remove("testJSON.json")
	if err := RenderConfig(testJSON, "file://testJSON.json"); err != nil {
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
	if err := RenderConfig(testJSON, "-"); err != nil {
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
	"backends": [{
					"name": "upstreamA{{.TESTRENDERCONFIGISPARSEABLE}}",
					"poll": 11,
					"onChange": "/bin/to/onChangeEvent/for/upstream/A.sh"}]}`

	os.Setenv("TESTRENDERCONFIGISPARSEABLE", "-ok")
	template, _ := renderConfigTemplate(testJSON)
	config, err := LoadConfig(string(template))
	if err != nil {
		t.Fatalf("unexpected error in LoadConfig: %v", err)
	}
	name := config.Watches[0].Name
	if name != "upstreamA-ok" {
		t.Fatalf("expected Watches[0] name to be upstreamA-ok but got %s", name)
	}
}

// Helper Functions

func validateParseEnvironment(t *testing.T, message string, environ []string, expected Environment) {
	if parsed := parseEnvironment(environ); !reflect.DeepEqual(expected, parsed) {
		t.Fatalf("%s; Expected %s but got %s", message, expected, parsed)
	}
}

func validateTemplate(t *testing.T, name string, template string, env Environment, expected string) {
	tmpl, err := NewTemplate([]byte(template))
	if err != nil {
		t.Fatalf("%s - Error parsing template: %s", name, err)
	}
	tmpl.Env = env
	res, err2 := tmpl.Execute()
	if err2 != nil {
		t.Fatalf("%s - Error executing template: %s", name, err2)
	}
	strRes := string(res)
	if strRes != expected {
		t.Fatalf("%s - Expected %s but got: %s", name, expected, strRes)
	}
}

func testRenderExpectError(t *testing.T, testJSON, render, expected string) {
	err := RenderConfig(testJSON, render)
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %s but got %s", expected, err)
	}
}

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
