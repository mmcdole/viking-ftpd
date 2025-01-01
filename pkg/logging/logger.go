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
	accessLog *log.Logger
	errorLog  *log.Logger
)

// Initialize sets up logging with the given configuration
func Initialize(config *Config) error {
	// Always set up error logger to stderr
	errorLog = log.New(os.Stderr, "ERROR: ", log.LstdFlags)

	// Set up access logger
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
		accessLog = log.New(logFile, "", 0)
	} else {
		accessLog = log.New(io.Discard, "", 0)
	}

	return nil
}

// LogAccess logs an FTP operation in a consistent format
func LogAccess(operation, user, path, status string, details ...interface{}) {
	// Skip logging for common read operations that succeeded
	if status == "success" {
		switch operation {
		case "SIZE", "MDTM", "LIST_DIR":
			return
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] user=%s operation=%s", timestamp, user, operation)
	
	if path != "" {
		msg += fmt.Sprintf(" path=%s", path)
	}
	
	msg += fmt.Sprintf(" status=%s", status)

	// Add any extra details
	for i := 0; i < len(details)-1; i += 2 {
		if key, ok := details[i].(string); ok {
			msg += fmt.Sprintf(" %s=%v", key, details[i+1])
		}
	}

	accessLog.Println(msg)
}

// LogError logs unexpected system errors to stderr
func LogError(operation string, err error, details ...interface{}) {
	// Extract user and path from details
	var user, path string
	for i := 0; i < len(details)-1; i += 2 {
		if key, ok := details[i].(string); ok {
			if key == "user" && i+1 < len(details) {
				user = fmt.Sprint(details[i+1])
			} else if key == "path" && i+1 < len(details) {
				path = fmt.Sprint(details[i+1])
			}
		}
	}

	// Handle expected errors as regular access logs
	if os.IsNotExist(err) {
		switch operation {
		case "SIZE", "MDTM", "DOWNLOAD":
			LogAccess(operation, user, path, "not_found")
			return
		}
	}
	if os.IsPermission(err) {
		LogAccess(operation, user, path, "denied")
		return
	}

	// Log unexpected errors to stderr
	msg := fmt.Sprintf("%s failed: %v", operation, err)
	if user != "" {
		msg += fmt.Sprintf(" (user: %s)", user)
	}
	if path != "" {
		msg += fmt.Sprintf(" (path: %s)", path)
	}
	errorLog.Println(msg)
}
