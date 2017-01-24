package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// A Render Application
type RenderApp struct {
	ConfigFlag     string
	RenderFlag     string
	RenderedConfig []byte
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

	a.ConfigFlag = configFlag
	a.RenderFlag = renderFlag

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

	a.RenderedConfig = template

	return a, nil
}

func (a *RenderApp) Run() {
	if a.RenderFlag == "-" {
		fmt.Printf("%s", a.RenderedConfig)
		os.Exit(0)
	}
	if strings.HasPrefix(a.RenderFlag, "file://") {
		var err error
		fName := strings.SplitAfter(a.RenderFlag, "file://")[1]
		if err = ioutil.WriteFile(fName, a.RenderedConfig, 0644); err != nil {
			panic(fmt.Errorf("Could not write config file: %s", err))
		}
		os.Exit(0)
	}
	os.Exit(1)
}
