package m3da

import (
	"log"
	"os"
)

// Logger interface allows users to provide their own logging implementation
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// defaultLogger is a simple logger implementation that writes to stderr
type defaultLogger struct {
	debugEnabled bool
	infoEnabled  bool
	warnEnabled  bool
	errorEnabled bool
}

// NewDefaultLogger creates a default logger with configurable levels
func NewDefaultLogger(debugEnabled, infoEnabled, warnEnabled, errorEnabled bool) Logger {
	return &defaultLogger{
		debugEnabled: debugEnabled,
		infoEnabled:  infoEnabled,
		warnEnabled:  warnEnabled,
		errorEnabled: errorEnabled,
	}
}

// NewQuietLogger creates a logger that only logs errors
func NewQuietLogger() Logger {
	return NewDefaultLogger(false, false, false, true)
}

// NewDebugLogger creates a logger that logs everything including debug messages
func NewDebugLogger() Logger {
	return NewDefaultLogger(true, true, true, true)
}

func (l *defaultLogger) Debug(format string, args ...interface{}) {
	if l.debugEnabled {
		log.Printf("[DEBUG] "+format, args...)
	}
}

func (l *defaultLogger) Info(format string, args ...interface{}) {
	if l.infoEnabled {
		log.Printf("[INFO] "+format, args...)
	}
}

func (l *defaultLogger) Warn(format string, args ...interface{}) {
	if l.warnEnabled {
		log.Printf("[WARN] "+format, args...)
	}
}

func (l *defaultLogger) Error(format string, args ...interface{}) {
	if l.errorEnabled {
		log.Printf("[ERROR] "+format, args...)
	}
}

// noOpLogger is a logger that does nothing
type noOpLogger struct{}

func (l *noOpLogger) Debug(format string, args ...interface{}) {}
func (l *noOpLogger) Info(format string, args ...interface{})  {}
func (l *noOpLogger) Warn(format string, args ...interface{})  {}
func (l *noOpLogger) Error(format string, args ...interface{}) {}

// NewNoOpLogger creates a logger that doesn't log anything
func NewNoOpLogger() Logger {
	return &noOpLogger{}
}

// Global logger instance - users can replace this with their own implementation
var globalLogger Logger = NewQuietLogger()

// SetLogger allows users to set their own logger implementation
func SetLogger(logger Logger) {
	globalLogger = logger
}

// EnableDebugLogging enables debug logging using the default logger
func EnableDebugLogging() {
	SetLogger(NewDebugLogger())
}

// DisableAllLogging disables all logging
func DisableAllLogging() {
	SetLogger(NewNoOpLogger())
}

// Internal logging functions used throughout the library
func debugf(format string, args ...interface{}) {
	globalLogger.Debug(format, args...)
}

func infof(format string, args ...interface{}) {
	globalLogger.Info(format, args...)
}

func warnf(format string, args ...interface{}) {
	globalLogger.Warn(format, args...)
}

func errorf(format string, args ...interface{}) {
	globalLogger.Error(format, args...)
}

// isDebugEnabled checks if we should check for environment variables on initialization
func init() {
	// Check for M3DA_DEBUG environment variable
	if os.Getenv("M3DA_DEBUG") == "1" || os.Getenv("M3DA_DEBUG") == "true" {
		EnableDebugLogging()
	}
}
