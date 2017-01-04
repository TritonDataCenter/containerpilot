package config

import (
	"reflect"
	"testing"
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
