package client

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

type HTTPClient struct {
	http.Client
	socketPath   string
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

func (c HTTPClient) Reload() error {
	resp, err := c.Post("http://control/v3/reload", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mesg := fmt.Sprintf("expected 200 but got %v\n%+v", resp.StatusCode, resp)
		return errors.New(mesg)
	}

	return nil
}

// func (c Client) SetMaintenance()
// func (c Client) PutEnv()
// func (c Client) PutMetric()
