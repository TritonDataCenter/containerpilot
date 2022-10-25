// Package subcommands provides all the alternative top-level functions
// that run one-off commands that don't run the main event loop.
package subcommands

import (
	"encoding/json"
	"fmt"

	"github.com/tritondatacenter/containerpilot/client"
	"github.com/tritondatacenter/containerpilot/config"
)

// Params ...
type Params struct {
	Version string
	GitHash string

	ConfigPath      string
	RenderFlag      string
	MaintenanceFlag string

	Metrics map[string]string
	Env     map[string]string
}

// Handler functions implement a subcommand
type Handler func(Params) error

// VersionHandler prints the version info only
func VersionHandler(params Params) error {
	fmt.Printf("Version: %s\nGitHash: %s\n", params.Version, params.GitHash)
	return nil
}

// RenderHandler asks the configuration package to render the
// configuration to the path provided
func RenderHandler(params Params) error {
	return config.RenderConfig(params.ConfigPath, params.RenderFlag)
}

// ReloadHandler fires a Reload request through the HTTPClient.
func ReloadHandler(params Params) error {
	client, err := initClient(params.ConfigPath)
	if err != nil {
		return err
	}
	if err := client.Reload(); err != nil {
		return fmt.Errorf("-reload: failed to run subcommand: %v", err)
	}
	return nil
}

// MaintenanceHandler fires either an enable or disable SetMaintenance
// request through the HTTPClient.
func MaintenanceHandler(params Params) error {
	client, err := initClient(params.ConfigPath)
	if err != nil {
		return err
	}
	flag := false
	if params.MaintenanceFlag == "enable" {
		flag = true
	}
	if err := client.SetMaintenance(flag); err != nil {
		return fmt.Errorf("-maintenance: failed to run subcommand: %v", err)
	}
	return nil
}

// PutEnvHandler fires a PutEnv request through the HTTPClient.
func PutEnvHandler(params Params) error {
	client, err := initClient(params.ConfigPath)
	if err != nil {
		return err
	}
	envJSON, err := json.Marshal(params.Env)
	if err != nil {
		return err
	}
	if err = client.PutEnv(string(envJSON)); err != nil {
		return fmt.Errorf("-reload: failed to run subcommand: %v", err)
	}
	return nil
}

// PutMetricsHandler fires a PutMetric request through the HTTPClient.
func PutMetricsHandler(params Params) error {
	client, err := initClient(params.ConfigPath)
	if err != nil {
		return err
	}
	metricsJSON, err := json.Marshal(params.Metrics)
	if err != nil {
		return err
	}
	if err = client.PutMetric(string(metricsJSON)); err != nil {
		return fmt.Errorf("-reload: failed to run subcommand: %v", err)
	}
	return nil
}

// GetPingHandler fires a ping check through the HTTPClient.
func GetPingHandler(params Params) error {
	client, err := initClient(params.ConfigPath)
	if err != nil {
		return err
	}
	if err := client.GetPing(); err != nil {
		return fmt.Errorf("-ping: failed: %v", err)
	}
	fmt.Println("ok")
	return nil
}

// loads the configuration so we can get the control socket and
// initializes the HTTPClient which callers will use for sending
// it commands
func initClient(configPath string) (*client.HTTPClient, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	httpclient, err := client.NewHTTPClient(cfg.Control.SocketPath)
	if err != nil {
		return nil, err
	}
	return httpclient, nil
}

// we're using an interface{} for params for Handler but these should
// never fail to type-assert. so this assert should be unreachable
// unless we've screwed something up.
//
// staticcheck U1000 func unused
//func assert(ok bool, msg string) {
//	if !ok {
//		panic("invalid parameter types for %v")
//	}
//}
