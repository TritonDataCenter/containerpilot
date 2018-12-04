// Package template provides rendering for the configuration files and
// the extra functions for use by the golang templating engine.
package template

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
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
		kv := strings.SplitN(e, "=", 2)
		env[kv[0]] = kv[1]
	}
	return env
}

func envFunc(env string) string {
	return os.Getenv(env)
}

func ensureInt(intv interface{}) (int, error) {
	switch intv.(type) {
	case string:
		ret, err := strconv.Atoi(intv.(string))
		if err != nil {
			return 0, err
		}
		return ret, nil
	default:
		return intv.(int), nil
	}
}

// loop accepts 1 or two parameters
// loop 5 returns 0 1 2 3 4 or loop 5 8 returns 5 6 7 or loop 5 1 returns 5 4 3 2
// loop also accepts a string or environment variable in the form of loop 0 .COUNT
func loop(params ...interface{}) ([]int, error) {
	var start, stop int
	result := []int{}

	switch len(params) {
	case 1:
		firstParam, err := ensureInt(params[0])
		if err != nil {
			return nil, err
		}
		start, stop = 0, firstParam
	case 2:
		firstParam, err := ensureInt(params[0])
		if err != nil {
			return nil, err
		}
		secondParam, err := ensureInt(params[1])
		if err != nil {
			return nil, err
		}
		start, stop = firstParam, secondParam
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

// Apply creates and renders a template from the given config template
func Apply(config []byte) ([]byte, error) {
	template, err := NewTemplate(config)
	if err != nil {
		return nil, err
	}
	return template.Execute()
}
