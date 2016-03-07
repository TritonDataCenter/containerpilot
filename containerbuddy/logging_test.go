package containerbuddy

import (
	"os"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

func TestLoggingBootstrap(t *testing.T) {
	defaultLog.init()
	std := logrus.StandardLogger()
	if std.Level != logrus.InfoLevel {
		t.Errorf("Expected INFO level logs, but got: %s", std.Level)
	}
	if std.Out != os.Stdout {
		t.Errorf("Expected output to Stdout")
	}
	if _, ok := std.Formatter.(*DefaultLogFormatter); !ok {
		t.Errorf("Expected *containerbuddy.DefaultLogFormatter got: %v", reflect.TypeOf(std.Formatter))
	}
}

func TestLoggingConfigInit(t *testing.T) {
	testLog := &LogConfig{
		Level:  "DEBUG",
		Format: "text",
		Output: "stderr",
	}
	testLog.init()
	std := logrus.StandardLogger()
	if std.Level != logrus.DebugLevel {
		t.Errorf("Expected 'debug' level logs, but got: %s", std.Level)
	}
	if std.Out != os.Stderr {
		t.Errorf("Expected output to Stderr")
	}
	if _, ok := std.Formatter.(*logrus.TextFormatter); !ok {
		t.Errorf("Expected *logrus.TextFormatter got: %v", reflect.TypeOf(std.Formatter))
	}
	// Reset to defaults
	defaultLog.init()
}