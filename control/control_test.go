package control

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/tritondatacenter/containerpilot/events"
	"github.com/tritondatacenter/containerpilot/tests"
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
	s.Bus = events.NewEventBus()
	s.Register(s.Bus)

	if err != nil {
		t.Fatalf("Could not init control server: %s", err)
	}
	return s
}

func TestNewHTTPServer(t *testing.T) {
	s := SetupHTTPServer(t, `{}`)
	defer os.Remove(DefaultSocket)
	assert.Equal(t, s.Addr, DefaultSocket, "expected server addr to ref default socket")

	tempSocketPath := tempSocketPath()
	defer os.Remove(tempSocketPath)
	s = SetupHTTPServer(t, fmt.Sprintf(`{ "socket": %q }`, tempSocketPath))
	assert.Equal(t, s.Addr, tempSocketPath, "expected server addr to ref default socket")
}

func TestValidate(t *testing.T) {
	srv := &HTTPServer{
		Addr: "",
	}
	if err := srv.Validate(); assert.NotNil(t, err) {
		assert.Equal(t, ErrMissingAddr, err, "expected missing addr error")
	}

	socketPath := tempSocketPath()
	srv = &HTTPServer{
		Addr: socketPath,
	}
	defer os.Remove(socketPath)
	if _, err := os.Create(socketPath); err != nil {
		assert.Nil(t, err, "expected test socket to be created")
	}
	if err := srv.Validate(); assert.Nil(t, err) {
		_, err := os.Stat(socketPath)
		assert.True(t, os.IsNotExist(err), "expected test socket to no longer exist")
	}
}

func TestServerSmokeTest(t *testing.T) {
	tempSocketPath := tempSocketPath()
	defer os.Remove(tempSocketPath)
	_, cancel := context.WithCancel(context.Background())

	s := SetupHTTPServer(t, fmt.Sprintf(`{ "socket": %q}`, tempSocketPath))
	defer s.Stop()
	s.Start(cancel)

	client := &http.Client{
		Transport: &http.Transport{
			Dial: socketDialer(tempSocketPath),
		},
	}

	// note the host name 'control' is meaningless here but the client
	// requires it for the connection string
	resp, err := client.Get("http://control/v3/xxxx")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 but got %v\n%+v", resp.StatusCode, resp)
	}
}
