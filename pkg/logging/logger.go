package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Config holds logging configuration
type Config struct {
	AccessLogPath string // Path to access log file
	ErrorLogPath  string // Path to error log file
}

var (
	access *log.Logger
	errors *log.Logger
)

// Initialize sets up logging with the given configuration
func Initialize(config *Config) error {
	// Ensure log directories exist
	if err := os.MkdirAll(filepath.Dir(config.AccessLogPath), 0755); err != nil {
		return fmt.Errorf("creating access log directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(config.ErrorLogPath), 0755); err != nil {
		return fmt.Errorf("creating error log directory: %w", err)
	}

	// Open log files
	accessFile, err := os.OpenFile(config.AccessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening access log: %w", err)
	}
	errorFile, err := os.OpenFile(config.ErrorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		accessFile.Close()
		return fmt.Errorf("opening error log: %w", err)
	}

	access = log.New(accessFile, "", 0)
	errors = log.New(errorFile, "", 0)

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
	
	access.Println(msg)
}

// LogError logs a system error in the format:
// [2025-01-01 15:04:05] ERROR operation=STOR error="permission denied" details=...
func LogError(operation string, err error, details ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf("[%s] ERROR operation=%s error=%q", timestamp, operation, err)
	
	if len(details) > 0 {
		msg += fmt.Sprintf(" details=%v", details)
	}
	
	errors.Println(msg)
}
