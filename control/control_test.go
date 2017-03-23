package control

import (
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

var jsonFragment = []byte(`{ "message": true }`)

// func TestNewControlServer(t *testing.T) {
// 	if s, err := NewServer(nil); err != nil {
// 		t.Fatalf("Could not init control server: %s", err)
// 	} else {
// 		// initial server
// 		s.Serve()
// 		assertServerIsActive(t, s)
// 		s.Shutdown()

// 		// reloaded server
// 		s, err := NewServer(nil)
// 		if err != nil {
// 			t.Fatalf("Could not init control server: %s", err)
// 		}
// 		s.Serve()
// 		assertServerIsActive(t, s)
// 	}
// }

func assertServerIsActive(t *testing.T, s *Server) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	assertGetStatusHandled(t, s)
}

func dialSocket(proto, path string) (conn net.Conn, err error) {
	return net.Dial("unix", "/var/run/containerpilot.sock")
}

func assertGetStatusHandled(t *testing.T, s *Server) {
	transport := &http.Transport{
		Dial: dialSocket,
	}
	client := &http.Client{
		Transport: transport,
	}

	// 'control' means nothing, connection string must use protocol/url format
	resp, err := client.Get("http://control/status")
	if err != nil {
		t.Fatal(err)
	}

	output, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		t.Logf("resp.Body: %s\n", output)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("Got %v status from control server", resp.StatusCode)
	}
}
