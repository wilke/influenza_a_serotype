package pafprocessor

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	// LogLevelDebug is the debug log level
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is the info log level
	LogLevelInfo
	// LogLevelWarn is the warn log level
	LogLevelWarn
	// LogLevelError is the error log level
	LogLevelError
)

// LoggerConfig represents the configuration for the logger
type LoggerConfig struct {
	Level      LogLevel
	OutputPath string
}

// LoggerInstance represents a logger instance
type LoggerInstance struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	level       LogLevel
}

// Logger is the global logger instance
var Logger *LoggerInstance

// InitLogger initializes the global logger
func InitLogger(config LoggerConfig) error {
	var output io.Writer = os.Stdout

	if config.OutputPath != "" {
		err := os.MkdirAll(filepath.Dir(config.OutputPath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.OutputPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		output = io.MultiWriter(os.Stdout, file)
	}

	debugLogger := log.New(output, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	infoLogger := log.New(output, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	warnLogger := log.New(output, "WARN: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	errorLogger := log.New(output, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)

	Logger = &LoggerInstance{
		debugLogger: debugLogger,
		infoLogger:  infoLogger,
		warnLogger:  warnLogger,
		errorLogger: errorLogger,
		level:       config.Level,
	}

	return nil
}

// Debugf logs a debug message
func (l *LoggerInstance) Debugf(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.debugLogger.Printf(format, args...)
	}
}

// Infof logs an info message
func (l *LoggerInstance) Infof(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.infoLogger.Printf(format, args...)
	}
}

// Warnf logs a warning message
func (l *LoggerInstance) Warnf(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.warnLogger.Printf(format, args...)
	}
}

// Errorf logs an error message
func (l *LoggerInstance) Errorf(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.errorLogger.Printf(format, args...)
	}
}

// ParseLogLevel parses a log level string
func ParseLogLevel(level string) (LogLevel, error) {
	switch level {
	case "debug":
		return LogLevelDebug, nil
	case "info":
		return LogLevelInfo, nil
	case "warn":
		return LogLevelWarn, nil
	case "error":
		return LogLevelError, nil
	default:
		return LogLevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// Performance metrics
var (
	performanceMetrics = make(map[string]time.Duration)
	performanceMutex   = &sync.Mutex{}
)

// LogPerformance logs the performance of a function
func LogPerformance(name string, startTime time.Time) {
	duration := time.Since(startTime)

	performanceMutex.Lock()
	defer performanceMutex.Unlock()

	performanceMetrics[name] = duration

	if Logger != nil {
		Logger.Debugf("%s took %s", name, duration)
	}
}

// GetPerformanceMetrics returns the performance metrics
func GetPerformanceMetrics() map[string]time.Duration {
	performanceMutex.Lock()
	defer performanceMutex.Unlock()

	metrics := make(map[string]time.Duration)
	for k, v := range performanceMetrics {
		metrics[k] = v
	}

	return metrics
}

// PrintPerformanceMetrics prints the performance metrics
func PrintPerformanceMetrics() {
	metrics := GetPerformanceMetrics()

	if Logger != nil {
		Logger.Infof("Performance Metrics:")
		for name, duration := range metrics {
			Logger.Infof("  %s: %s", name, duration)
		}
	}
}
