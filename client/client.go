package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type HTTPClient struct {
	http.Client
	socketPath  string
}

var SocketType = "unix"

func socketDialer(socketPath string) func(string, string) (net.Conn, error) {
	return func(_, _ string) (net.Conn, error) {
		return net.Dial(SocketType, socketPath)
	}
}

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

func (self HTTPClient) Reload() (*http.Response, error) {
	resp, err := self.Post("http://control/v3/reload", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mesg := fmt.Sprintf("expected 200 but got %v\n%+v", resp.StatusCode, resp)
		return nil, errors.New(mesg)
	}

	return resp, nil
}

func (self HTTPClient) SetMaintenance(isEnabled bool) (*http.Response, error) {
	var flag string
	if isEnabled {
		flag = "enable"
	} else {
		flag = "disable"
	}

	resp, err := self.Post("http://control/v3/maintenance/" + flag, "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mesg := fmt.Sprintf("expected 200 but got %v\n%+v", resp.StatusCode, resp)
		return nil, errors.New(mesg)
	}

	return resp, nil
}

func (self HTTPClient) PutEnv(body string) (*http.Response, error) {
	resp, err := self.Post("http://control/v3/env", "application/json",
		strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mesg := fmt.Sprintf("expected 200 but got %v\n%+v", resp.StatusCode, resp)
		return nil, errors.New(mesg)
	}

	return resp, nil
}

func (self HTTPClient) PutMetric(body string) (*http.Response, error) {
	resp, err := self.Post("http://control/v3/metric", "application/json",
		strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mesg := fmt.Sprintf("expected 200 but got %v\n%+v", resp.StatusCode, resp)
		return nil, errors.New(mesg)
	}

	return resp, nil
}
