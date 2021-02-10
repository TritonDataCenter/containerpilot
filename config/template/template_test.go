package template

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEnvironment(t *testing.T) {
	parsed := parseEnvironment([]string{})
	assert.Equal(t, Environment{}, parsed)

	parsed = parseEnvironment([]string{
		"VAR1=test",
		"VAR2=test2",
		"VAR3=test3=w/equals==",
	})
	assert.Equal(t, Environment{
		"VAR1": "test",
		"VAR2": "test2",
		"VAR3": "test3=w/equals==",
	}, parsed)
}

func TestTemplate(t *testing.T) {
	env := parseEnvironment([]string{
		"NAME=Template",
		"USER=pilot",
		"PARTS=a:b:c",
		"COUNT=3",
	})
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
	testTemplate("Loop string", `{{ loop .COUNT }}`, "[0 1 2]")
	testTemplate("Loop string range", `{{ loop 1 .COUNT }}`, "[1 2]")
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
