package utils

import (
	"bufio"
	"io"

	log "github.com/Sirupsen/logrus"
)

// NewLogWriter pipes stdout/err logs to logrus
func NewLogWriter(fields log.Fields, level log.Level) io.WriteCloser {
	r, w := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			entry := log.WithFields(fields)
			entry.Level = level
			entry.Println(scanner.Text())
		}
	}()
	return w
}
