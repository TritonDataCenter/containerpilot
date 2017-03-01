package socket

import (
	// "encoding/json"
	// "fmt"
	"net"
	"net/http"
	// "strings"
	"testing"
)

var jsonFragment = []byte(`{ "message": true }`)

func TestNewControlSocket(t *testing.T) {
	if cs, err := NewControlSocket(nil); err != nil {
		t.Fatalf("Could not init control socket: %s", err)
	} else {
		// initial server
		cs.Serve()
		assertSocketIsActive(t, cs)
		cs.Shutdown()

		// reloaded server
		cs, err := NewControlSocket(nil)
		if err != nil {
			t.Fatalf("Could not init control socket: %s", err)
		}
		cs.Serve()
		assertSocketIsActive(t, cs)
	}
}

func assertSocketIsActive(t *testing.T, cs *ControlSocket) {
	cs.lock.RLock()
	defer cs.lock.RUnlock()
	assertGetStatusHandled(t, cs)
}

func dialSocket(proto, path string) (conn net.Conn, err error) {
	return net.Dial("unix", "/var/run/containerpilot.sock")
}

func assertGetStatusHandled(t *testing.T, cs *ControlSocket) {
	transport := &http.Transport{
		Dial: dialSocket,
	}
	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get("http://sock/stats")
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("Got %v status from control socket", resp.StatusCode)
	}

}
