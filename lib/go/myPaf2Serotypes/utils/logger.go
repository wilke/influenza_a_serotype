// Package utils provides utility functions and types for myPaf2Serotypes
package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger interface defines logging methods
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
}

// Field represents a logging field
type Field struct {
	Key   string
	Value interface{}
}

// ZeroLogger wraps zerolog.Logger to implement our Logger interface
type ZeroLogger struct {
	logger zerolog.Logger
}

// NewLogger creates a new logger instance
func NewLogger(level, format string, debug bool) Logger {
	var zLevel zerolog.Level
	switch level {
	case "debug":
		zLevel = zerolog.DebugLevel
	case "info":
		zLevel = zerolog.InfoLevel
	case "warn":
		zLevel = zerolog.WarnLevel
	case "error":
		zLevel = zerolog.ErrorLevel
	default:
		zLevel = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(zLevel)

	var logger zerolog.Logger
	if format == "json" {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		// Pretty console output
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(output).With().Timestamp().Logger()
	}

	// Add caller information in debug mode
	if debug {
		logger = logger.With().Caller().Logger()
	}

	return &ZeroLogger{logger: logger}
}

// Debug logs a debug level message
func (l *ZeroLogger) Debug(msg string, fields ...Field) {
	event := l.logger.Debug()
	for _, f := range fields {
		event = addField(event, f)
	}
	event.Msg(msg)
}

// Info logs an info level message
func (l *ZeroLogger) Info(msg string, fields ...Field) {
	event := l.logger.Info()
	for _, f := range fields {
		event = addField(event, f)
	}
	event.Msg(msg)
}

// Warn logs a warning level message
func (l *ZeroLogger) Warn(msg string, fields ...Field) {
	event := l.logger.Warn()
	for _, f := range fields {
		event = addField(event, f)
	}
	event.Msg(msg)
}

// Error logs an error level message
func (l *ZeroLogger) Error(msg string, fields ...Field) {
	event := l.logger.Error()
	for _, f := range fields {
		event = addField(event, f)
	}
	event.Msg(msg)
}

// Fatal logs a fatal level message and exits the program
func (l *ZeroLogger) Fatal(msg string, fields ...Field) {
	event := l.logger.Fatal()
	for _, f := range fields {
		event = addField(event, f)
	}
	event.Msg(msg)
}

// With creates a child logger with additional fields
func (l *ZeroLogger) With(fields ...Field) Logger {
	newLogger := l.logger.With()
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			newLogger = newLogger.Str(f.Key, v)
		case int:
			newLogger = newLogger.Int(f.Key, v)
		case int64:
			newLogger = newLogger.Int64(f.Key, v)
		case float64:
			newLogger = newLogger.Float64(f.Key, v)
		case bool:
			newLogger = newLogger.Bool(f.Key, v)
		case error:
			newLogger = newLogger.Err(v)
		case time.Time:
			newLogger = newLogger.Time(f.Key, v)
		case time.Duration:
			newLogger = newLogger.Dur(f.Key, v)
		default:
			newLogger = newLogger.Interface(f.Key, v)
		}
	}
	return &ZeroLogger{logger: newLogger.Logger()}
}

// WithContext creates a child logger with context
func (l *ZeroLogger) WithContext(ctx context.Context) Logger {
	// Extract trace ID from context if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		return l.With(Field{Key: "trace_id", Value: traceID})
	}
	return l
}

// Helper function to add a field to an event
func addField(event *zerolog.Event, f Field) *zerolog.Event {
	switch v := f.Value.(type) {
	case string:
		return event.Str(f.Key, v)
	case int:
		return event.Int(f.Key, v)
	case int64:
		return event.Int64(f.Key, v)
	case float64:
		return event.Float64(f.Key, v)
	case bool:
		return event.Bool(f.Key, v)
	case error:
		if f.Key == "error" {
			return event.Err(v)
		}
		return event.Str(f.Key, v.Error())
	case time.Time:
		return event.Time(f.Key, v)
	case time.Duration:
		return event.Dur(f.Key, v)
	default:
		return event.Interface(f.Key, v)
	}
}

// Helper functions for creating fields

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Time creates a time field
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// LoggerKey is the context key for storing logger
type loggerKey struct{}

// ContextWithLogger returns a new context with the logger attached
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFromContext retrieves the logger from context
func LoggerFromContext(ctx context.Context) (Logger, bool) {
	logger, ok := ctx.Value(loggerKey{}).(Logger)
	return logger, ok
}

// GetLogger retrieves logger from context or returns a default logger
func GetLogger(ctx context.Context) Logger {
	if logger, ok := LoggerFromContext(ctx); ok {
		return logger
	}
	// Return a default logger if none in context
	return NewLogger("info", "text", false)
}

// Performance logging helpers

// LogDuration logs the duration of an operation
func LogDuration(logger Logger, operation string, start time.Time, fields ...Field) {
	duration := time.Since(start)
	allFields := append(fields,
		String("operation", operation),
		Duration("duration", duration),
	)
	logger.Info(fmt.Sprintf("%s completed", operation), allFields...)
}

// LogError logs an error with context
func LogError(logger Logger, operation string, err error, fields ...Field) {
	allFields := append(fields,
		String("operation", operation),
		Err(err),
	)
	logger.Error(fmt.Sprintf("%s failed", operation), allFields...)
}