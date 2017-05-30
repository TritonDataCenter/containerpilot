package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
)

// Environment is a map of environment variables to their values
type Environment map[string]string

// split is a version of strings.Split that can be piped
func split(sep, s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}, nil
	}
	return strings.Split(s, sep), nil
}

// join is a version of strings.Join that can be piped
func join(sep string, s []string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}
	return strings.Join(s, sep), nil
}

// replaceAll replaces all occurrences of a value in a string with the given
// replacement value.
func replaceAll(from, to, s string) (string, error) {
	return strings.Replace(s, from, to, -1), nil
}

// regexReplaceAll replaces all occurrences of a regex in a string with the given
// replacement value.
func regexReplaceAll(re, to, s string) (string, error) {
	compiled, err := regexp.Compile(re)
	if err != nil {
		return "", err
	}
	return compiled.ReplaceAllString(s, to), nil
}

func parseEnvironment(environ []string) Environment {
	env := make(Environment)
	if len(environ) == 0 {
		return env
	}
	for _, e := range environ {
		kv := strings.Split(e, "=")
		env[kv[0]] = kv[1]
	}
	return env
}

func envFunc(env string) string {
	return os.Getenv(env)
}

// loop accepts 1 or two parameters
// loop 5 returns 0 1 2 3 4 or loop 5 8 returns 5 6 7 or loop 5 1 returns 5 4 3 2
func loop(params ...int) ([]int, error) {
	var start, stop int
	result := []int{}

	switch len(params) {
	case 1:
		start, stop = 0, params[0]
	case 2:
		start, stop = params[0], params[1]
	default:
		return nil, fmt.Errorf("loop: wrong number of arguments, expected 1 or 2"+
			", but got %d", len(params))
	}

	if stop < start {
		for i := start; i > stop; i-- {
			result = append(result, i)
		}
	} else {
		for i := start; i < stop; i++ {
			result = append(result, i)

		}
	}
	return result, nil
}

// Template encapsulates a golang template
// and its associated environment variables.
type Template struct {
	Template *template.Template
	Env      Environment
}

func defaultValue(defaultValue, templateValue interface{}) string {
	if templateValue != nil {
		if str, ok := templateValue.(string); ok && str != "" {
			return str
		}
	}
	defaultStr, ok := defaultValue.(string)
	if !ok {
		return fmt.Sprintf("%v", defaultValue)
	}
	return defaultStr
}

// NewTemplate creates a Template parsed from the configuration
// and the current environment variables
func NewTemplate(config []byte) (*Template, error) {
	env := parseEnvironment(os.Environ())
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"default":         defaultValue,
		"env":             envFunc,
		"split":           split,
		"join":            join,
		"replaceAll":      replaceAll,
		"regexReplaceAll": regexReplaceAll,
		"loop":            loop,
	}).Option("missingkey=zero").Parse(string(config))
	if err != nil {
		return nil, err
	}
	return &Template{
		Env:      env,
		Template: tmpl,
	}, nil
}

// Execute renders the template
func (c *Template) Execute() ([]byte, error) {
	var buffer bytes.Buffer
	if err := c.Template.Execute(&buffer, c.Env); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ApplyTemplate creates and renders a template from the given config template
func ApplyTemplate(config []byte) ([]byte, error) {
	template, err := NewTemplate(config)
	if err != nil {
		return nil, err
	}
	return template.Execute()
}
