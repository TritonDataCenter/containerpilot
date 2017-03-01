package socket

import (
	// "encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

type ControlSocket struct {
	Path string
	mux *http.ServeMux
	addr net.UnixAddr
	listening bool
	lock sync.RWMutex
}

// Handle `/status` control endpoint
// NOTE: Stubbed out temporarily....
func GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{ "message": true }`))
}

// Initialize a new ControlSocket for our process
func NewControlSocket(raw interface{}) (*ControlSocket, error) {
	// TODO: Take from config or CLI args
	// TODO: Add default file perms (?)
	cs := &ControlSocket{
		Path: "/var/run/containerpilot.sock",
		listening: false,
	}

	// NOTE: Might not need this...
	if err := utils.DecodeRaw(raw, cs); err != nil {
		return nil, fmt.Errorf("Control socket configuration error: %v", err)
	}

	cs.addr = net.UnixAddr{
		Name: cs.Path,
		Net: "unix",
	}

	cs.mux = http.NewServeMux()
	cs.mux.Handle("/status", http.HandlerFunc(GetStatusHandler))
	return cs, nil
}

var listener net.Listener

// Serve our present ControlSocket
func (cs *ControlSocket) Serve() {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	// ref https://github.com/joyent/containerpilot/pull/165
	if listener != nil {
		return
	}

	ln, err := net.Listen(cs.addr.Network(), cs.addr.String())
	if err != nil {
		log.Fatalf("Error serving socket at %s: %v", cs.addr.String(), err)
	}

	listener = ln
	cs.listening = true

	go func() {
		log.Infof("socket: Listening at %s", cs.addr.String())
		log.Fatal(http.Serve(ln, cs.mux))
		log.Debugf("socket: Stopped listening at %s", cs.addr.String())
	}()
}

// Shutdown shuts down the telemetry service
func (cs *ControlSocket) Shutdown() {
	log.Debug("socket: shutdown received but currently a no-op")
}
