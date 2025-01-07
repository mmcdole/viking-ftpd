package logging

import "fmt"

// LogLevel represents the severity of a log message
type LogLevel string

const (
	// LogLevelDebug is for debug messages
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo is for informational messages
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn is for warning messages
	LogLevelWarn LogLevel = "warn"
	// LogLevelError is for error messages
	LogLevelError LogLevel = "error"
	// LogLevelPanic is for panic messages
	LogLevelPanic LogLevel = "panic"
)

// Initialize sets up the global loggers
func Initialize(accessLogPath, appLogPath string, level LogLevel) error {
	var err error

	// Set default level if not specified
	if level == "" {
		level = LogLevelInfo
	}

	// Initialize access logger first since it's simpler
	Access, err = NewAccessLogger(accessLogPath)
	if err != nil {
		return fmt.Errorf("failed to initialize access logger: %w", err)
	}

	// Initialize application logger
	App, err = NewAppLogger(appLogPath, level)
	if err != nil {
		return fmt.Errorf("failed to initialize app logger: %w", err)
	}

	return nil
}

// MustInitialize initializes logging and panics on error
func MustInitialize(accessLogPath, appLogPath string, level LogLevel) {
	if err := Initialize(accessLogPath, appLogPath, level); err != nil {
		panic(fmt.Sprintf("failed to initialize logging: %v", err))
	}
}

var (
	// App is the global application logger
	App *AppLogger
	// Access is the global access logger
	Access AccessLogger
)
