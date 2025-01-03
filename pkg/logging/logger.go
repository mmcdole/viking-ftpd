// Package logging provides structured logging for the FTP server
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Operation represents filesystem operations
type Operation string

const (
	OpOpen    Operation = "OPEN"    // Opening a file for reading
	OpCreate  Operation = "CREATE"  // Creating or opening a file for writing
	OpMkdir   Operation = "MKDIR"   // Creating a directory
	OpRemove  Operation = "REMOVE"  // Removing a file or directory
	OpDelete  Operation = "DELETE"  // Deleting a file or directory
	OpRename  Operation = "RENAME"  // Renaming a file or directory
	OpReadDir Operation = "READDIR" // Reading directory contents
	OpAuth    Operation = "AUTH"    // Authentication attempt
)

// Mode represents file access mode
type Mode string

const (
	ModeRead  Mode = "READ"
	ModeWrite Mode = "WRITE"
)

// Entry represents a log entry
type Entry struct {
	Operation Operation
	User      string
	Path      string
	Mode      Mode
	FromPath  string    // For rename operations
	ToPath    string    // For rename operations
	Entries   int       // For readdir operations
	IP        string    // For auth operations
	Error     error
	Time      time.Time
}

// Config holds logging configuration
type Config struct {
	AccessLogPath string // Path to access log file
	ErrorLogPath  string // Path to error log file
}

// Logger handles FTP server logging
type Logger struct {
	access *log.Logger
	error  *log.Logger
}

// New creates a new Logger
func New(config Config) (*Logger, error) {
	var accessWriter io.Writer = os.Stdout
	if config.AccessLogPath != "" {
		if err := os.MkdirAll(filepath.Dir(config.AccessLogPath), 0755); err != nil {
			return nil, fmt.Errorf("create access log directory: %w", err)
		}
		f, err := os.OpenFile(config.AccessLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open access log: %w", err)
		}
		accessWriter = f
	}

	var errorWriter io.Writer = os.Stderr
	if config.ErrorLogPath != "" {
		if err := os.MkdirAll(filepath.Dir(config.ErrorLogPath), 0755); err != nil {
			return nil, fmt.Errorf("create error log directory: %w", err)
		}
		f, err := os.OpenFile(config.ErrorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("open error log: %w", err)
		}
		errorWriter = f
	}

	return &Logger{
		access: log.New(accessWriter, "", 0),
		error:  log.New(errorWriter, "", 0),
	}, nil
}

// formatMessage formats a log entry
func formatMessage(e Entry) string {
	var msg strings.Builder

	// Time and operation (left-padded to 7 chars)
	msg.WriteString(fmt.Sprintf("%s %-7s", e.Time.Format("2006-01-02 15:04:05"), e.Operation))

	// User
	msg.WriteString(fmt.Sprintf(" User: '%s'", e.User))

	// Path (for most operations)
	if e.Path != "" && e.Operation != OpRename {
		msg.WriteString(fmt.Sprintf(" Path: '%s'", e.Path))
	}

	// Mode (only for OPEN)
	if e.Operation == OpOpen && e.Mode != "" {
		msg.WriteString(fmt.Sprintf(" Mode: %s", e.Mode))
	}

	// Special fields for specific operations
	switch e.Operation {
	case OpRename:
		msg.WriteString(fmt.Sprintf(" From: '%s' To: '%s'", e.FromPath, e.ToPath))
	case OpReadDir:
		msg.WriteString(fmt.Sprintf(" Entries: %d", e.Entries))
	case OpAuth:
		msg.WriteString(fmt.Sprintf(" IP: '%s'", e.IP))
	}

	// Status
	if e.Error != nil {
		msg.WriteString(fmt.Sprintf(" [FAILURE] Error: %s", e.Error))
	} else {
		msg.WriteString(" [SUCCESS]")
	}

	return msg.String()
}

// Log writes a log entry
func (l *Logger) Log(e Entry) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	msg := formatMessage(e)
	if e.Error != nil {
		l.error.Println(msg)
	} else {
		l.access.Println(msg)
	}
}

// LogOpen logs file open operations
func (l *Logger) LogOpen(user, path string, mode Mode, err error) {
	l.Log(Entry{
		Operation: OpOpen,
		User:      user,
		Path:      path,
		Mode:      mode,
		Error:     err,
	})
}

// LogCreate logs file create operations
func (l *Logger) LogCreate(user, path string, err error) {
	l.Log(Entry{
		Operation: OpCreate,
		User:      user,
		Path:      path,
		Error:     err,
	})
}

// LogMkdir logs directory creation operations
func (l *Logger) LogMkdir(user, path string, err error) {
	l.Log(Entry{
		Operation: OpMkdir,
		User:      user,
		Path:      path,
		Error:     err,
	})
}

// LogRemove logs file/directory removal operations
func (l *Logger) LogRemove(user, path string, err error) {
	l.Log(Entry{
		Operation: OpRemove,
		User:      user,
		Path:      path,
		Error:     err,
	})
}

// LogDelete logs file/directory deletion operations
func (l *Logger) LogDelete(user, path string, err error) {
	l.Log(Entry{
		Operation: OpDelete,
		User:      user,
		Path:      path,
		Error:     err,
		Time:      time.Now(),
	})
}

// LogRename logs file/directory rename operations
func (l *Logger) LogRename(user, fromPath, toPath string, err error) {
	l.Log(Entry{
		Operation: OpRename,
		User:      user,
		FromPath:  fromPath,
		ToPath:    toPath,
		Error:     err,
	})
}

// LogReadDir logs directory listing operations
func (l *Logger) LogReadDir(user, path string, entries int, err error) {
	l.Log(Entry{
		Operation: OpReadDir,
		User:      user,
		Path:      path,
		Entries:   entries,
		Error:     err,
	})
}

// LogAuth logs authentication attempts
func (l *Logger) LogAuth(user, ip string, err error) {
	l.Log(Entry{
		Operation: OpAuth,
		User:      user,
		IP:        ip,
		Error:     err,
	})
}

// Package level functions that use defaultLogger
var defaultLogger *Logger

// Initialize sets up the default logger
func Initialize(config *Config) error {
	if config == nil {
		config = &Config{}
	}
	logger, err := New(*config)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// Default logger functions
func LogOpen(user, path string, mode Mode, err error)              { defaultLogger.LogOpen(user, path, mode, err) }
func LogCreate(user, path string, err error)                       { defaultLogger.LogCreate(user, path, err) }
func LogMkdir(user, path string, err error)                       { defaultLogger.LogMkdir(user, path, err) }
func LogRemove(user, path string, err error)                      { defaultLogger.LogRemove(user, path, err) }
func LogDelete(user, path string, err error)                      { defaultLogger.LogDelete(user, path, err) }
func LogRename(user, fromPath, toPath string, err error)          { defaultLogger.LogRename(user, fromPath, toPath, err) }
func LogReadDir(user, path string, entries int, err error)        { defaultLogger.LogReadDir(user, path, entries, err) }
func LogAuth(user, ip string, err error)                          { defaultLogger.LogAuth(user, ip, err) }
