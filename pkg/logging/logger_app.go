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
	writer *RotatingWriter // nil if logging to stdout
}

// NewAppLogger creates a new application logger
func NewAppLogger(logPath string, level LogLevel, maxSize int64, verifyInterval time.Duration) (*AppLogger, error) {
	var writer io.Writer = os.Stdout
	var rotatingWriter *RotatingWriter

	if logPath != "" {
		rw, err := NewRotatingWriter(logPath, maxSize, verifyInterval)
		if err != nil {
			return nil, fmt.Errorf("creating rotating writer: %w", err)
		}
		writer = rw
		rotatingWriter = rw
	}

	return &AppLogger{
		level:  level,
		logger: log.New(writer, "", 0), // No flags, we'll handle formatting ourselves
		writer: rotatingWriter,
	}, nil
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

// Close closes the logger and stops background rotation
func (l *AppLogger) Close() error {
	if l.writer != nil {
		return l.writer.Close()
	}
	return nil
}
