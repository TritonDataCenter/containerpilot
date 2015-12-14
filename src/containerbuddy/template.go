package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

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
	if defaultStr, ok := defaultValue.(string); !ok {
		return fmt.Sprintf("%v", defaultValue)
	} else {
		return defaultStr
	}
}

// Interpolate variables
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

func (c *ConfigTemplate) Execute() ([]byte, error) {
	var buffer bytes.Buffer
	if err := c.Template.Execute(&buffer, c.Env); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func ApplyTemplate(config []byte) ([]byte, error) {
	template, err := NewConfigTemplate(config)
	if err != nil {
		return nil, err
	}
	return template.Execute()
}
