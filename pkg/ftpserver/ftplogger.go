package ftpserver

import (
	"log"
	"os"
	golog "github.com/fclairamb/go-log"
)

// FTPLogger implements the ftpserverlib logger interface
type FTPLogger struct {
	logger *log.Logger
}

// NewFTPLogger creates a new FTP logger that writes to stdout
func NewFTPLogger() *FTPLogger {
	return &FTPLogger{
		logger: log.New(os.Stdout, "FTP: ", log.LstdFlags),
	}
}

// Debug logs a debug message
func (l *FTPLogger) Debug(message string, keyvals ...interface{}) {
	l.logger.Printf("DEBUG: %s %v", message, keyvals)
}

// Info logs an info message
func (l *FTPLogger) Info(message string, keyvals ...interface{}) {
	l.logger.Printf("INFO: %s %v", message, keyvals)
}

// Warn logs a warning message
func (l *FTPLogger) Warn(message string, keyvals ...interface{}) {
	l.logger.Printf("WARN: %s %v", message, keyvals)
}

// Error logs an error message
func (l *FTPLogger) Error(message string, keyvals ...interface{}) {
	l.logger.Printf("ERROR: %s %v", message, keyvals)
}

// Panic logs a panic message
func (l *FTPLogger) Panic(message string, keyvals ...interface{}) {
	l.logger.Printf("PANIC: %s %v", message, keyvals)
}

// With adds context to the logger
func (l *FTPLogger) With(keyvals ...interface{}) golog.Logger {
	// For stdout logging, we'll just return the same logger
	return l
}
