// Package logger manages the configuration of logging
package logger

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/client9/reopen"
	"github.com/sirupsen/logrus"
)

// Config configures the log levels
type Config struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

var defaultLog = &Config{
	Level:  "INFO",
	Format: "default",
	Output: "stdout",
}

func init() {
	if err := defaultLog.Init(); err != nil {
		log.Println(err)
	}
}

// Init initializes the logger and sets default values if not provided
func (l *Config) Init() error {
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
		formatter = &logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		}
	case "default":
		formatter = &DefaultLogFormatter{
			TimestampFormat: time.RFC3339Nano,
		}
	default:
		return fmt.Errorf("Unknown log format '%s'", l.Format)
	}
	switch strings.ToLower(l.Output) {
	case "stderr":
		output = os.Stderr
	case "stdout":
		output = os.Stdout
	case "":
		return fmt.Errorf("Unknown output type '%s'", l.Output)
	default:
		f, err := reopen.NewFileWriter(l.Output)
		if err != nil {
			return fmt.Errorf("Error initializing log file '%s': %s", l.Output, err)
		}
		initializeSignal(f)
		output = f
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(formatter)
	logrus.SetOutput(output)
	return nil
}

// DefaultLogFormatter delegates formatting to standard go log package
type DefaultLogFormatter struct {
	TimestampFormat string
}

// Format formats the logrus entry by passing it to the "log" package
func (f *DefaultLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}
	logger := log.New(b, "", 0)
	logger.Println(time.Now().Format(f.TimestampFormat) + " " + string(entry.Message))
	// Panic and Fatal are handled by logrus automatically
	return b.Bytes(), nil
}

func initializeSignal(f *reopen.FileWriter) {
	// Handle SIGUSR1
	//
	// channel is number of signals needed to catch  (more or less)
	// we only are working with one here, SIGUSR1
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGUSR1)
	go func() {
		for {
			<-sighup
			f.Reopen()
		}
	}()
}
