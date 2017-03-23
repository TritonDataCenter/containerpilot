package control

import (
	"strings"
	"testing"

	"github.com/joyent/containerpilot/tests"
)

func TestControlConfigDefault(t *testing.T) {
	cfg, err := NewConfig(nil)
	if err != nil {
		t.Fatalf("could not parse control config JSON: %s", err)
	}

	if strings.Compare(cfg.Socket, DEFAULT_SOCKET) != 0 {
		t.Fatalf("test socket does not match parsed socket")
	}
}

func TestControlConfigParse(t *testing.T) {
	testSocket := "/var/run/cp3.sock"
	testRaw := tests.DecodeRaw(`{ "socket": "/var/run/cp3.sock" }`)
	if testRaw == nil {
		t.Fatalf("parsed empty control config JSON")
	}

	cfg, err := NewConfig(testRaw)
	if err != nil {
		t.Fatalf("could not parse control config JSON: %s", err)
	}

	if strings.Compare(cfg.Socket, testSocket) != 0 {
		t.Fatalf("test socket does not match parsed socket")
	}
}

