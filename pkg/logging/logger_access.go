package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
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
		logger: log.New(writer, "", 0), // No flags, we'll handle formatting ourselves
	}, nil
}

// formatValue formats a value for logfmt, quoting if necessary
func formatValue(v interface{}) string {
	s := fmt.Sprintf("%v", v)
	// Quote if contains space, equals, or quotes
	if strings.ContainsAny(s, " =\"") {
		// Escape existing quotes
		s = strings.ReplaceAll(s, "\"", "\\\"")
		return fmt.Sprintf("\"%s\"", s)
	}
	return s
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
