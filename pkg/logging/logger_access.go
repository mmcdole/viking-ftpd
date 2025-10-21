package logging

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"
)

// AccessLogger defines the interface for FTP operation logging
type AccessLogger interface {
	// LogAccess logs FTP operations
	LogAccess(operation string, user string, path string, status string, details ...interface{})
	// LogAuth logs authentication operations
	LogAuth(operation string, user string, status string, details ...interface{})
	// Close closes the logger and stops background rotation
	Close() error
}

type accessLogger struct {
	logger *log.Logger
	writer *RotatingWriter // nil if logging to io.Discard
}

// NewAccessLogger creates a new access logger
func NewAccessLogger(logPath string, maxSize int64, verifyInterval time.Duration) (AccessLogger, error) {
	var writer io.Writer
	var rotatingWriter *RotatingWriter

	if logPath == "" {
		writer = io.Discard
	} else {
		rw, err := NewRotatingWriter(logPath, maxSize, verifyInterval)
		if err != nil {
			return nil, fmt.Errorf("creating rotating writer: %w", err)
		}
		writer = rw
		rotatingWriter = rw
	}

	return &accessLogger{
		logger: log.New(writer, "", 0), // No flags, we'll handle formatting ourselves
		writer: rotatingWriter,
	}, nil
}

func (l *accessLogger) LogAccess(operation string, user string, path string, status string, details ...interface{}) {
	var parts []string
	parts = append(parts, fmt.Sprintf("op=%s", formatValue(operation)))
	if user != "" {
		parts = append(parts, fmt.Sprintf("user=%s", formatValue(user)))
	}
	if path != "" {
		parts = append(parts, fmt.Sprintf("path=%s", formatValue(path)))
	}
	parts = append(parts, fmt.Sprintf("status=%s", formatValue(status)))

	for i := 0; i < len(details); i += 2 {
		if i+1 < len(details) {
			parts = append(parts, fmt.Sprintf("%v=%s", details[i], formatValue(details[i+1])))
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 -0700")
	l.logger.Printf("%s %s", timestamp, strings.Join(parts, " "))
}

func (l *accessLogger) LogAuth(operation string, user string, status string, details ...interface{}) {
	var parts []string
	parts = append(parts, fmt.Sprintf("op=%s", formatValue(operation)))
	if user != "" {
		parts = append(parts, fmt.Sprintf("user=%s", formatValue(user)))
	}
	parts = append(parts, fmt.Sprintf("status=%s", formatValue(status)))

	for i := 0; i < len(details); i += 2 {
		if i+1 < len(details) {
			parts = append(parts, fmt.Sprintf("%v=%s", details[i], formatValue(details[i+1])))
		}
	}

	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 -0700")
	l.logger.Printf("%s %s", timestamp, strings.Join(parts, " "))
}

// Close closes the logger and stops background rotation
func (l *accessLogger) Close() error {
	if l.writer != nil {
		return l.writer.Close()
	}
	return nil
}
