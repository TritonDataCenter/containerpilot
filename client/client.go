// Package client provides a HTTP client used to send commands to the
// ContainerPilot control socket
package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// HTTPClient provides a properly configured http.Client object used to send
// requests out to a ContainerPilot process's control socket.
type HTTPClient struct {
	http.Client
	socketPath string
}

var socketType = "unix"

func socketDialer(socketPath string) func(string, string) (net.Conn, error) {
	return func(_, _ string) (net.Conn, error) {
		return net.Dial(socketType, socketPath)
	}
}

// NewHTTPClient initializes an client.HTTPClient object by configuring it's
// socketPath for HTTP communication through the local file system.
func NewHTTPClient(socketPath string) (*HTTPClient, error) {
	if socketPath == "" {
		err := errors.New("control server not loading due to missing config")
		return nil, err
	}

	client := &HTTPClient{}
	client.Transport = &http.Transport{
		Dial: socketDialer(socketPath),
	}

	return client, nil
}

// Reload makes a request to the reload endpoint of a ContainerPilot process.
func (c HTTPClient) Reload() error {
	resp, err := c.Post("http://control/v3/reload", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SetMaintenance makes a request to either the enable or disable maintenance
// endpoint of a ContainerPilot process.
func (c HTTPClient) SetMaintenance(isEnabled bool) error {
	flag := "disable"
	if isEnabled {
		flag = "enable"
	}

	resp, err := c.Post("http://control/v3/maintenance/"+flag, "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// PutEnv makes a request to the environ endpoint of a ContainerPilot process
// for setting environ variable pairs.
func (c HTTPClient) PutEnv(body string) error {
	resp, err := c.Post("http://control/v3/environ", "application/json",
		strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("unprocessable entity received by control server")
	}
	return nil
}

// PutMetric makes a request to the metric endpoint of a ContainerPilot process
// for setting custom metrics.
func (c HTTPClient) PutMetric(body string) error {
	resp, err := c.Post("http://control/v3/metric", "application/json",
		strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("unprocessable entity received by control server")
	}
	return nil
}

// GetPing make a request to the ping endpoint of the ContainerPilot control
// socket, to verify it's listening
func (c HTTPClient) GetPing() error {
	resp, err := c.Get("http://control/v3/ping")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return fmt.Errorf("unprocessable entity received by control server")
	}
	return nil
}
