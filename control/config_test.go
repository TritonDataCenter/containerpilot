package control

import (
	"strings"
	"testing"

	"github.com/tritondatacenter/containerpilot/tests"
)

func TestControlConfigDefault(t *testing.T) {
	cfg, err := NewConfig(nil)
	if err != nil {
		t.Fatalf("could not parse control config JSON: %s", err)
	}

	if strings.Compare(cfg.SocketPath, DefaultSocket) != 0 {
		t.Fatal("parsed socket does not match default socket")
	}
}

func TestControlConfigParse(t *testing.T) {
	testSocket := "/var/run/cp3.sock"
	testRaw := tests.DecodeRaw(`{ "socket": "/var/run/cp3.sock" }`)
	if testRaw == nil {
		t.Fatal("parsed empty control config JSON")
	}

	cfg, err := NewConfig(testRaw)
	if err != nil {
		t.Fatalf("could not parse control config JSON: %s", err)
	}

	if strings.Compare(cfg.SocketPath, testSocket) != 0 {
		t.Fatal("parsed socket does not match custom socket")
	}
}
