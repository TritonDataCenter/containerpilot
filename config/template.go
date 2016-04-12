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

// ConfigTemplate encapsulates a golang template
// and its associated environment variables.
type ConfigTemplate struct {
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

// NewConfigTemplate creates a ConfigTemplate parsed from the configuration
// and the current environment variables
func NewConfigTemplate(config []byte) (*ConfigTemplate, error) {
	env := parseEnvironment(os.Environ())
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"default": defaultValue,
	}).Option("missingkey=zero").Parse(string(config))
	if err != nil {
		return nil, err
	}
	return &ConfigTemplate{
		Env:      env,
		Template: tmpl,
	}, nil
}

// Execute renders the template
func (c *ConfigTemplate) Execute() ([]byte, error) {
	var buffer bytes.Buffer
	if err := c.Template.Execute(&buffer, c.Env); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ApplyTemplate creates and renders a template from the given config template
func ApplyTemplate(config []byte) ([]byte, error) {
	template, err := NewConfigTemplate(config)
	if err != nil {
		return nil, err
	}
	return template.Execute()
}
