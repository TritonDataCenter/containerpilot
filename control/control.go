package control

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/joyent/containerpilot/utils"
)

// Server contains the state of the HTTP Server used by ContainerPilot's HTTP
// transport control plane. Currently this is listening via a UNIX socket file.
type Server struct {
	Socket    string      `mapstructure:"socket"`
	Mux       *http.ServeMux
	addr      net.UnixAddr
	listening bool
	lock      sync.RWMutex
}

// NewServer initializes a new control server for manipulating ContainerPilot's
// runtime configuration.
func NewServer(raw interface{}) (*Server, error) {
	// TODO: Take from config or CLI args
	// TODO: Add default file perms (?)
	s := &Server{
		Socket: "/var/run/containerpilot.sock",
		listening: false,
	}

	if err := utils.DecodeRaw(raw, s); err != nil {
		return nil, fmt.Errorf("Control server configuration error: %v", err)
	}

	s.addr = net.UnixAddr{
		Name: s.Socket,
		Net: "unix",
	}

	s.Mux = http.NewServeMux()
	return s, nil
}

var listener net.Listener

// Serve starts serving the control server
func (s *Server) Serve() {
	s.lock.Lock()
	defer s.lock.Unlock()

	// ref https://github.com/joyent/containerpilot/pull/165
	if listener != nil {
		return
	}

	ln, err := net.Listen(s.addr.Network(), s.addr.String())
	if err != nil {
		log.Fatalf("Error serving socket at %s: %v", s.addr.String(), err)
	}

	listener = ln
	s.listening = true

	go func() {
		log.Infof("control: Serving at %s", s.addr.String())
		log.Fatal(http.Serve(ln, s.Mux))
		log.Debugf("control: Stopped serving at %s", s.addr.String())
	}()
}

// Shutdown shuts down the control server
func (s *Server) Shutdown() {
	s.listening = false
	log.Debug("control: Shutdown received but currently a no-op")
}
