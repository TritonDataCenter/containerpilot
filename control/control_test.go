package control

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joyent/containerpilot/events"
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
	s.Bus = events.NewEventBus()
	s.Subscribe(s.Bus)

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
