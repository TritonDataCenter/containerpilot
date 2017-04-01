package control

import (
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
)

func dialSocket(proto, path string) (conn net.Conn, err error) {
	return net.Dial(SocketType, DefaultSocket)
}

func SetupHTTPServer(t *testing.T, raw string) *HTTPServer {
	testRaw := tests.DecodeRaw(raw)
	cfg, err := NewConfig(testRaw)
	if err != nil {
		t.Fatalf("parsed empty control config JSON")
	}

	s, err := NewHTTPServer(cfg)
	if err != nil {
		t.Fatalf("Could not init control server: %s", err)
	}

	return s
}

func TestNewHTTPServer(t *testing.T) {
	s := SetupHTTPServer(t, `{}`)
	assert.False(t, s.listening)
	assert.Equal(t, s.addr.Name, DefaultSocket, "expected server addr to ref default socket")
	assert.Equal(t, s.addr.Net, SocketType, "expected server addr to ref socket type")
}

func TestGetEnv(t *testing.T) {
	os.Setenv("FLIP_MODE", "dangerous")
	defer os.Unsetenv("FLIP_MODE")
	testBody := "FLIP_MODE=dangerous"

	s := SetupHTTPServer(t, `{}`)
	defer s.Shutdown()
	s.Serve()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialSocket,
		},
	}

	// NOTE: 'control' means nothing here, connection string must use
	// protocol/url format.
	resp, err := client.Get("http://control/env")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
	}

	output, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(output), testBody)  {
		t.Fatalf("HTTP response should include %v", testBody)
	}
}

func TestGetEnvAsPost(t *testing.T) {
	os.Setenv("FLIP_MODE", "dangerous")
	defer os.Unsetenv("FLIP_MODE")
	testBody := "FLIP_MODE=dangerous"

	s := SetupHTTPServer(t, `{}`)
	defer s.Shutdown()
	s.Serve()

	client := &http.Client{
		Transport: &http.Transport{
			Dial: dialSocket,
		},
	}

	r := strings.NewReader("{}\n")
	// NOTE: 'control' means nothing here, connection string must use
	// protocol/url format.
	resp, err := client.Post("http://control/env", "application/json", r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("HTTP response should not return status %v\n%+v", resp.StatusCode, resp)
	}

	output, err := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(output), testBody) {
		t.Fatalf("HTTP response should not include %v", testBody)
	}
}
