package control

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joyent/containerpilot/tests"
	"github.com/joyent/containerpilot/tests/assert"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func socketDialer(tempSocketPath string) func(string, string) (net.Conn, error) {
	return func(_, _ string) (net.Conn, error) {
		return net.Dial(SocketType, tempSocketPath)
	}
}

func tempSocketPath() string {
	filename := fmt.Sprintf("containerpilot-test-socket-%d", rand.Int())
	return filepath.Join(os.TempDir(), filename)
}

func SetupHTTPServer(t *testing.T, raw string) *HTTPServer {
	testRaw := tests.DecodeRaw(raw)
	cfg, err := NewConfig(testRaw)
	if err != nil {
		t.Fatal("parsed empty control config JSON")
	}

	s, err := NewHTTPServer(cfg)
	if err != nil {
		t.Fatalf("Could not init control server: %s", err)
	}

	return s
}

func TestNewHTTPServer(t *testing.T) {
	s := SetupHTTPServer(t, `{}`)
	// assert.False(t, s.listening, "expected listening to be false")
	assert.Equal(t, s.Addr, DefaultSocket, "expected server addr to ref default socket")
	// assert.Equal(t, s.addr.Net, SocketType, "expected server addr to ref socket type")
}

func TestGetEnv(t *testing.T) {
	os.Setenv("FLIP_MODE", "dangerous")
	defer os.Unsetenv("FLIP_MODE")
	testBody := "FLIP_MODE=dangerous"

	tempSocketPath := tempSocketPath()
	defer os.Remove(tempSocketPath)

	s := SetupHTTPServer(t, fmt.Sprintf(`{ "socket": %q}`, tempSocketPath))
	defer s.Stop()
	s.Start(nil)

	client := &http.Client{
		Transport: &http.Transport{
			Dial: socketDialer(tempSocketPath),
		},
	}

	t.Run("GET", func(t *testing.T) {
		// NOTE: 'control' means nothing here, connection string must use
		// protocol/url format.
		resp, err := client.Get("http://control/v3/env")
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

		if !strings.Contains(string(output), testBody) {
			t.Fatalf("HTTP response should include %v", testBody)
		}
	})

	t.Run("POST", func(t *testing.T) {
		r := strings.NewReader("{}\n")
		// NOTE: 'control' means nothing here, connection string must use
		// protocol/url format.
		resp, err := client.Post("http://control/v3/env", "application/json", r)
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
	})
}
