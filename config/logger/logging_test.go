package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestLoggingBootstrap(t *testing.T) {
	defaultLog.Init()
	std := logrus.StandardLogger()
	if std.Level != logrus.InfoLevel {
		t.Errorf("Expected INFO level logs, but got: %s", std.Level)
	}
	if std.Out != os.Stdout {
		t.Errorf("Expected output to Stdout")
	}
	if _, ok := std.Formatter.(*DefaultLogFormatter); !ok {
		t.Errorf("Expected *containerpilot.DefaultLogFormatter got: %v", reflect.TypeOf(std.Formatter))
	}
}

func TestLoggingConfigInit(t *testing.T) {
	testLog := &Config{
		Level:  "DEBUG",
		Format: "text",
		Output: "stderr",
	}
	testLog.Init()
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
	defaultLog.Init()
}

func TestDefaultFormatterEmptyMessage(t *testing.T) {
	formatter := &DefaultLogFormatter{}
	_, err := formatter.Format(logrus.WithFields(
		logrus.Fields{
			"level": "info",
			"msg":   "something",
		},
	))
	if err != nil {
		t.Errorf("Did not expect error: %v", err)
	}
}

func TestDefaultFormatterPanic(t *testing.T) {
	defaultLog.Init()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic but did not")
		}
	}()
	logrus.Panicln("Panic Test")
}

func TestFileLogger(t *testing.T) {
	// initialize logger
	filename := "/tmp/test_log"
	testLog := &Config{
		Level:  "DEBUG",
		Format: "text",
		Output: filename,
	}
	err := testLog.Init()
	if err != nil {
		t.Errorf("Did not expect error: %v", err)
	}

	testContainsLog := func(fname string, log string, t *testing.T) {
		content, err := ioutil.ReadFile(fname)
		if err != nil {
			t.Errorf("Did not expect error: %v", err)
		}
		if len(content) == 0 {
			t.Error("could not write log to file")
		}
		logs := string(content)
		if !strings.Contains(logs, log) {
			t.Errorf("expected log file to contain '%s', got '%s'", log, logs)
		}
	}

	// write a log message
	logMsg := "this is a test"
	logrus.Info(logMsg)
	testContainsLog(filename, logMsg, t)

	// rotate the log
	rotatedFilename := fmt.Sprintf("%s.1", filename)
	err = os.Rename(filename, rotatedFilename)
	if err != nil {
		t.Errorf("Did not expect error: %v", err)
	}
	logMsg = "this is another test"
	logrus.Info(logMsg)
	testContainsLog(rotatedFilename, logMsg, t)

	// reopen file
	syscall.Kill(os.Getpid(), syscall.SIGUSR1)
	time.Sleep(1*time.Second)

	timeExpire := time.Now().Add(3 * time.Second)
	for {
		_, err = os.Stat(filename)
		if err == nil {
			break
		}
		if timeExpire.After(time.Now()) {
			t.Error("Did not reopen file in expected time")
			return
		}
		time.Sleep(1 * time.Second)
	}
	logMsg = "this is the last test"
	logrus.Info(logMsg)
	testContainsLog(filename, logMsg, t)
}
