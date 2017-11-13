package discovery

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// TestServer represents a Consul server we can run our tests against. Depends
// on a local `consul` binary installed into our environ's PATH.
type TestServer struct {
	cmd      *exec.Cmd
	HTTPAddr string
}

// NewTestServer constructs a new TestServer by including the httpPort as well.
func NewTestServer(httpPort int) (*TestServer, error) {
	path, err := exec.LookPath("consul")
	if err != nil || path == "" {
		return nil, fmt.Errorf("consul not found on $PATH - download and install " +
			"consul or skip this test")
	}

	args := []string{"agent", "-dev"}
	cmd := exec.Command("consul", args...)
	cmd.Stdout = io.Writer(os.Stdout)
	cmd.Stderr = io.Writer(os.Stderr)
	if err := cmd.Start(); err != nil {
		return nil, errors.New("failed starting command")
	}

	httpAddr := fmt.Sprintf("127.0.0.1:%d", httpPort)

	return &TestServer{
		cmd:      cmd,
		HTTPAddr: httpAddr,
	}, nil
}

// Stop stops a TestServer
func (s *TestServer) Stop() error {
	if s.cmd == nil {
		return nil
	}

	if s.cmd.Process != nil {
		if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
			return errors.New("failed to kill consul server")
		}
	}

	return s.cmd.Wait()
}
