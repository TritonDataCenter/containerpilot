package config

import (
	"bytes"
	"fmt"
	"io"
	"time"
	"log"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// LogConfig configures the log levels
type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

var defaultLog = &LogConfig{
	Level:  "INFO",
	Format: "default",
	Output: "stdout",
}

func init() {
	if err := defaultLog.init(); err != nil {
		log.Println(err)
	}
}

func (l *LogConfig) init() error {
	// Set defaults
	if l.Level == "" {
		l.Level = defaultLog.Level
	}
	if l.Format == "" {
		l.Format = defaultLog.Format
	}
	if l.Output == "" {
		l.Output = defaultLog.Output
	}
	level, err := logrus.ParseLevel(strings.ToLower(l.Level))
	if err != nil {
		return fmt.Errorf("Unknown log level '%s': %s", l.Level, err)
	}
	var formatter logrus.Formatter
	var output io.Writer
	switch strings.ToLower(l.Format) {
	case "text":
		formatter = &logrus.TextFormatter{}
	case "json":
		formatter = &logrus.JSONFormatter{}
	case "default":
		formatter = &DefaultLogFormatter{}
	default:
		return fmt.Errorf("Unknown log format '%s'", l.Format)
	}
	switch strings.ToLower(l.Output) {
	case "stderr":
		output = os.Stderr
	case "stdout":
		output = os.Stdout
	default:
		return fmt.Errorf("Unknown output type '%s'", l.Output)
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(formatter)
	logrus.SetOutput(output)
	return nil
}

// DefaultLogFormatter delegates formatting to standard go log package
type DefaultLogFormatter struct {
}

// RFC3339logWriter io.Writer that outputs RFC3339 dates
type RFC3339logWriter struct {
}

func (writer RFC3339logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().Format(time.RFC3339Nano) + " " + string(bytes))
}

// Format formats the logrus entry by passing it to the "log" package
func (f *DefaultLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	
	b := &bytes.Buffer{}
	logger := log.New(new(RFC3339logWriter), "", 0)
	logger.Println(entry.Message)
	// Panic and Fatal are handled by logrus automatically
	return b.Bytes(), nil
}
