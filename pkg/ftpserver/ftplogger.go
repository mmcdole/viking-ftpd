package ftpserver

import (
	"fmt"
	"github.com/fclairamb/go-log"
	"github.com/mmcdole/viking-ftpd/pkg/logging"
)

// FTPLogger wraps our custom logger to implement fclairamb/go-log interface
type FTPLogger struct {
	logger *logging.Logger
}

// NewFTPLogger creates a new FTP logger
func NewFTPLogger(logger *logging.Logger) *FTPLogger {
	return &FTPLogger{logger: logger}
}

// Debug logs a debug message
func (l *FTPLogger) Debug(message string, keyvals ...interface{}) {
	l.logger.Log(logging.Entry{
		Operation: logging.Operation(fmt.Sprintf("DEBUG: %s - %v", message, keyvals)),
	})
}

// Info logs an info message
func (l *FTPLogger) Info(message string, keyvals ...interface{}) {
	l.logger.Log(logging.Entry{
		Operation: logging.Operation(fmt.Sprintf("INFO: %s - %v", message, keyvals)),
	})
}

// Warn logs a warning message
func (l *FTPLogger) Warn(message string, keyvals ...interface{}) {
	l.logger.Log(logging.Entry{
		Operation: logging.Operation(fmt.Sprintf("WARN: %s - %v", message, keyvals)),
	})
}

// Error logs an error message
func (l *FTPLogger) Error(message string, keyvals ...interface{}) {
	l.logger.Log(logging.Entry{
		Operation: logging.Operation(fmt.Sprintf("ERROR: %s - %v", message, keyvals)),
		Error:     fmt.Errorf(message),
	})
}

// Panic logs a panic message
func (l *FTPLogger) Panic(message string, keyvals ...interface{}) {
	l.logger.Log(logging.Entry{
		Operation: logging.Operation(fmt.Sprintf("PANIC: %s - %v", message, keyvals)),
		Error:     fmt.Errorf(message),
	})
}

// With adds context to the logger
func (l *FTPLogger) With(keyvals ...interface{}) log.Logger {
	// Create a new logger with additional context
	return l
}
