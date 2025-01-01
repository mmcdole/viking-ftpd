package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Config holds logging configuration
type Config struct {
	AccessLogPath string // Path to access log file, optional
}

var (
	logger *log.Logger
)

// Initialize sets up logging with the given configuration
func Initialize(config *Config) error {
	// Set up logger
	if config.AccessLogPath != "" {
		// Ensure log directory exists
		if err := os.MkdirAll(filepath.Dir(config.AccessLogPath), 0755); err != nil {
			return fmt.Errorf("creating access log directory: %w", err)
		}

		// Open log file
		logFile, err := os.OpenFile(config.AccessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening access log: %w", err)
		}
		logger = log.New(logFile, "", 0)
	} else {
		// Use no-op logger if no path specified
		logger = log.New(io.Discard, "", 0)
	}

	return nil
}

// LogAccess logs an FTP operation in the format:
// [2025-01-01 15:04:05] user=alice operation=STOR path=/foo/bar.txt status=success size=1234
func LogAccess(operation, user, path, status string, details ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] user=%s operation=%s", timestamp, user, operation)

	if path != "" {
		msg += fmt.Sprintf(" path=%s", path)
	}

	msg += fmt.Sprintf(" status=%s", status)

	if len(details) > 0 {
		msg += fmt.Sprintf(" details=%v", details)
	}

	logger.Println(msg)
}

// LogError logs a system error in the format:
// [2025-01-01 15:04:05] ERROR operation=STOR error="permission denied" details=...
func LogError(operation string, err error, details ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] ERROR operation=%s error=%q", timestamp, operation, err)

	if len(details) > 0 {
		msg += fmt.Sprintf(" details=%v", details)
	}

	logger.Println(msg)
}
