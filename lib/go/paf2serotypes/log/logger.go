package log

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	// LevelDebug level for detailed troubleshooting
	LevelDebug LogLevel = iota
	// LevelInfo level for general operational information
	LevelInfo
	// LevelWarn level for warning conditions
	LevelWarn
	// LevelError level for error conditions
	LevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger represents a logger
type Logger struct {
	level  LogLevel
	writer io.Writer
	mu     sync.Mutex
}

// DefaultLogger is the default logger instance
var DefaultLogger = NewLogger(LevelInfo, os.Stdout)

// NewLogger creates a new logger
func NewLogger(level LogLevel, writer io.Writer) *Logger {
	return &Logger{
		level:  level,
		writer: writer,
	}
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetWriter sets the log writer
func (l *Logger) SetWriter(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = writer
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	message := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "[%s] %s: %s\n", timestamp, level.String(), message)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// LogMemoryUsage logs the current memory usage
func (l *Logger) LogMemoryUsage(operation string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	l.Info("%s: Memory usage - Alloc: %v MiB, TotalAlloc: %v MiB, Sys: %v MiB, NumGC: %v",
		operation,
		m.Alloc/1024/1024,
		m.TotalAlloc/1024/1024,
		m.Sys/1024/1024,
		m.NumGC)
}

// Debug logs a debug message to the default logger
func Debug(format string, args ...interface{}) {
	DefaultLogger.Debug(format, args...)
}

// Info logs an info message to the default logger
func Info(format string, args ...interface{}) {
	DefaultLogger.Info(format, args...)
}

// Warn logs a warning message to the default logger
func Warn(format string, args ...interface{}) {
	DefaultLogger.Warn(format, args...)
}

// Error logs an error message to the default logger
func Error(format string, args ...interface{}) {
	DefaultLogger.Error(format, args...)
}

// LogMemoryUsage logs the current memory usage to the default logger
func LogMemoryUsage(operation string) {
	DefaultLogger.LogMemoryUsage(operation)
}

// SetLevel sets the log level of the default logger
func SetLevel(level LogLevel) {
	DefaultLogger.SetLevel(level)
}

// SetWriter sets the log writer of the default logger
func SetWriter(writer io.Writer) {
	DefaultLogger.SetWriter(writer)
}

// ParseLogLevel parses a log level string
func ParseLogLevel(level string) (LogLevel, error) {
	switch level {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}
