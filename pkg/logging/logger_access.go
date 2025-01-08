package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// AccessLogger defines the interface for FTP operation logging
type AccessLogger interface {
	// LogAccess logs FTP operations
	LogAccess(operation string, user string, path string, status string, details ...interface{})
	// LogAuth logs authentication operations
	LogAuth(operation string, user string, status string, details ...interface{})
}

type accessLogger struct {
	logger *log.Logger
}

// NewAccessLogger creates a new access logger
func NewAccessLogger(logPath string) (AccessLogger, error) {
	var writer io.Writer

	if logPath == "" {
		writer = io.Discard
	} else {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening access log file: %w", err)
		}
		writer = f
	}

	return &accessLogger{
		logger: log.New(writer, "", log.LstdFlags),
	}, nil
}

func (l *accessLogger) LogAccess(operation string, user string, path string, status string, details ...interface{}) {
	var parts []string
	parts = append(parts, fmt.Sprintf("op=%s", operation))
	if user != "" {
		parts = append(parts, fmt.Sprintf("user=%s", user))
	}
	if path != "" {
		parts = append(parts, fmt.Sprintf("path=%s", path))
	}
	parts = append(parts, fmt.Sprintf("status=%s", status))

	for i := 0; i < len(details); i += 2 {
		if i+1 < len(details) {
			parts = append(parts, fmt.Sprintf("%v=%v", details[i], details[i+1]))
		}
	}

	l.logger.Print(strings.Join(parts, " "))
}

func (l *accessLogger) LogAuth(operation string, user string, status string, details ...interface{}) {
	var parts []string
	parts = append(parts, fmt.Sprintf("op=%s", operation))
	if user != "" {
		parts = append(parts, fmt.Sprintf("user=%s", user))
	}
	parts = append(parts, fmt.Sprintf("status=%s", status))

	for i := 0; i < len(details); i += 2 {
		if i+1 < len(details) {
			parts = append(parts, fmt.Sprintf("%v=%v", details[i], details[i+1]))
		}
	}

	l.logger.Print(strings.Join(parts, " "))
}
