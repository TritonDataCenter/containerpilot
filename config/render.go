package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// RenderApp encapsulates the rendered configuration file and the input and
// output paths.
type RenderApp struct {
	renderFlag     string
	renderedConfig []byte
}

// EmptyApp creates an empty application
func EmptyApp() *RenderApp {
	app := &RenderApp{}
	return app
}

// NewRenderApp creates a new Config Rendering application
func NewRenderApp(configFlag string, renderFlag string) (*RenderApp, error) {

	a := EmptyApp()

	if configFlag == "" {
		return nil, errors.New("-config flag is required")
	}
	if renderFlag != "-" && !strings.HasPrefix(renderFlag, "file://") {
		return nil, errors.New("-render flag is invalid")
	}

	a.renderFlag = renderFlag

	var data []byte
	if strings.HasPrefix(configFlag, "file://") {
		var err error
		fName := strings.SplitAfter(configFlag, "file://")[1]
		if data, err = ioutil.ReadFile(fName); err != nil {
			return nil, fmt.Errorf("Could not read config file: %s", err)
		}
	} else {
		data = []byte(configFlag)
	}

	template, err := ApplyTemplate(data)
	if err != nil {
		return nil, fmt.Errorf(
			"Could not apply template to config: %v", err)
	}

	a.renderedConfig = template

	return a, nil
}

// Run outputs the rendered configuration file to RenderApp.RenderFlag
func (a *RenderApp) Run() {
	if a.renderFlag == "-" {
		fmt.Printf("%s", a.renderedConfig)
		os.Exit(0)
	}
	if strings.HasPrefix(a.renderFlag, "file://") {
		var err error
		fName := strings.SplitAfter(a.renderFlag, "file://")[1]
		if err = ioutil.WriteFile(fName, a.renderedConfig, 0644); err != nil {
			panic(fmt.Errorf("Could not write config file: %s", err))
		}
		os.Exit(0)
	}
	os.Exit(1)
}
