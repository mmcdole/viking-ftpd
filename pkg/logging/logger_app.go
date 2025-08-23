package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	golog "github.com/fclairamb/go-log"
)

// AppLogger implements the go-log.Logger interface
type AppLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewAppLogger creates a new application logger
func NewAppLogger(logPath string, level LogLevel) (*AppLogger, error) {
	var writer io.Writer = os.Stdout
	if logPath != "" {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening app log file: %w", err)
		}
		writer = f
	}

	return &AppLogger{
		level:  level,
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

func (l *AppLogger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelPanic: 4,
	}
	return levels[level] >= levels[l.level]
}

func (l *AppLogger) log(level LogLevel, message string, keyvals ...interface{}) {
	if !l.shouldLog(level) {
		return
	}
	
	// Format key-value pairs
	var kvStrings []string
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			key := toString(keyvals[i])
			value := toString(keyvals[i+1])
			kvStrings = append(kvStrings, fmt.Sprintf("%s=%s", key, formatValue(value)))
		}
	}
	kvStr := strings.Join(kvStrings, " ")
	
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 -0700")
	l.logger.Printf("%s %s: %s %s", timestamp, level, message, kvStr)
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	
	str := fmt.Sprintf("%v", v)
	// Clean up the string
	str = strings.ReplaceAll(str, "\n", " ")
	str = strings.ReplaceAll(str, "\r", " ")
	str = strings.ReplaceAll(str, "\t", " ")
	// Collapse multiple spaces into one
	str = strings.Join(strings.Fields(str), " ")
	return str
}

// Debug implements go-log.Logger
func (l *AppLogger) Debug(message string, keyvals ...interface{}) {
	l.log(LogLevelDebug, message, keyvals...)
}

// Info implements go-log.Logger
func (l *AppLogger) Info(message string, keyvals ...interface{}) {
	l.log(LogLevelInfo, message, keyvals...)
}

// Warn implements go-log.Logger
func (l *AppLogger) Warn(message string, keyvals ...interface{}) {
	l.log(LogLevelWarn, message, keyvals...)
}

// Error implements go-log.Logger
func (l *AppLogger) Error(message string, keyvals ...interface{}) {
	l.log(LogLevelError, message, keyvals...)
}

// Panic implements go-log.Logger
func (l *AppLogger) Panic(message string, keyvals ...interface{}) {
	l.log(LogLevelPanic, message, keyvals...)
}

// With implements go-log.Logger
func (l *AppLogger) With(keyvals ...interface{}) golog.Logger {
	// For simplicity, we'll just return the same logger
	// In a more complex implementation, we could create a new logger with context
	return l
}

// IsDebug returns true if the logger is at debug level
func (l *AppLogger) IsDebug() bool {
	return l.level == LogLevelDebug
}
