package config

import (
	"bytes"
	"fmt"
	"os"
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
		"default": defaultValue,
		"split":   split,
		"join":    join,
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
