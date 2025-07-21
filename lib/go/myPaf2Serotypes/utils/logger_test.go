package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		format string
		debug  bool
	}{
		{
			name:   "JSON format info level",
			level:  "info",
			format: "json",
			debug:  false,
		},
		{
			name:   "Text format debug level",
			level:  "debug",
			format: "text",
			debug:  true,
		},
		{
			name:   "Invalid level defaults to info",
			level:  "invalid",
			format: "json",
			debug:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level, tt.format, tt.debug)
			assert.NotNil(t, logger)
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	// Create a buffer to capture log output
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	zl := &ZeroLogger{logger: logger}

	// Test Debug
	zl.Debug("debug message", String("key", "value"))
	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "value")
	buf.Reset()

	// Test Info
	zl.Info("info message", Int("count", 42))
	output = buf.String()
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "count")
	assert.Contains(t, output, "42")
	buf.Reset()

	// Test Warn
	zl.Warn("warning message", Bool("flag", true))
	output = buf.String()
	assert.Contains(t, output, "warning message")
	assert.Contains(t, output, "flag")
	assert.Contains(t, output, "true")
	buf.Reset()

	// Test Error
	zl.Error("error message", Float64("score", 0.95))
	output = buf.String()
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "score")
	assert.Contains(t, output, "0.95")
}

func TestLoggerWith(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	zl := &ZeroLogger{logger: logger}

	// Create child logger with additional fields
	childLogger := zl.With(
		String("component", "processor"),
		Int("worker_id", 1),
	)

	childLogger.Info("processing", String("file", "test.paf"))

	output := buf.String()
	assert.Contains(t, output, "component")
	assert.Contains(t, output, "processor")
	assert.Contains(t, output, "worker_id")
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "file")
	assert.Contains(t, output, "test.paf")
}

func TestLoggerWithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	zl := &ZeroLogger{logger: logger}

	// Test with trace ID in context
	ctx := context.WithValue(context.Background(), "trace_id", "abc123")
	ctxLogger := zl.WithContext(ctx)
	ctxLogger.Info("context message")

	output := buf.String()
	assert.Contains(t, output, "trace_id")
	assert.Contains(t, output, "abc123")
}

func TestFieldHelpers(t *testing.T) {
	// Test field creation helpers
	strField := String("name", "test")
	assert.Equal(t, "name", strField.Key)
	assert.Equal(t, "test", strField.Value)

	intField := Int("count", 10)
	assert.Equal(t, "count", intField.Key)
	assert.Equal(t, 10, intField.Value)

	int64Field := Int64("large", int64(1000000))
	assert.Equal(t, "large", int64Field.Key)
	assert.Equal(t, int64(1000000), int64Field.Value)

	floatField := Float64("score", 0.95)
	assert.Equal(t, "score", floatField.Key)
	assert.Equal(t, 0.95, floatField.Value)

	boolField := Bool("enabled", true)
	assert.Equal(t, "enabled", boolField.Key)
	assert.Equal(t, true, boolField.Value)

	errField := Err(assert.AnError)
	assert.Equal(t, "error", errField.Key)
	assert.Equal(t, assert.AnError, errField.Value)

	durField := Duration("elapsed", time.Second)
	assert.Equal(t, "elapsed", durField.Key)
	assert.Equal(t, time.Second, durField.Value)

	now := time.Now()
	timeField := Time("timestamp", now)
	assert.Equal(t, "timestamp", timeField.Key)
	assert.Equal(t, now, timeField.Value)

	anyField := Any("data", map[string]int{"a": 1})
	assert.Equal(t, "data", anyField.Key)
	assert.Equal(t, map[string]int{"a": 1}, anyField.Value)
}

func TestContextLogger(t *testing.T) {
	logger := NewLogger("info", "json", false)

	// Test adding logger to context
	ctx := ContextWithLogger(context.Background(), logger)
	
	// Test retrieving logger from context
	retrieved, ok := LoggerFromContext(ctx)
	assert.True(t, ok)
	assert.NotNil(t, retrieved)

	// Test retrieving from context without logger
	_, ok = LoggerFromContext(context.Background())
	assert.False(t, ok)

	// Test GetLogger with logger in context
	gotLogger := GetLogger(ctx)
	assert.NotNil(t, gotLogger)

	// Test GetLogger without logger in context (should return default)
	defaultLogger := GetLogger(context.Background())
	assert.NotNil(t, defaultLogger)
}

func TestLogDuration(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	zl := &ZeroLogger{logger: logger}

	start := time.Now().Add(-100 * time.Millisecond)
	LogDuration(zl, "test_operation", start, String("extra", "field"))

	output := buf.String()
	assert.Contains(t, output, "test_operation completed")
	assert.Contains(t, output, "operation")
	assert.Contains(t, output, "test_operation")
	assert.Contains(t, output, "duration")
	assert.Contains(t, output, "extra")
	assert.Contains(t, output, "field")
}

func TestLogError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf).With().Timestamp().Logger()
	zl := &ZeroLogger{logger: logger}

	err := assert.AnError
	LogError(zl, "test_operation", err, Int("retry", 3))

	output := buf.String()
	assert.Contains(t, output, "test_operation failed")
	assert.Contains(t, output, "operation")
	assert.Contains(t, output, "test_operation")
	assert.Contains(t, output, "error")
	assert.Contains(t, output, "retry")
	assert.Contains(t, output, "3")
}

func TestJSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf)
	zl := &ZeroLogger{logger: logger}

	zl.Info("test", String("key", "value"), Int("number", 42))

	// Parse JSON output
	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "test", result["message"])
	assert.Equal(t, "value", result["key"])
	assert.Equal(t, float64(42), result["number"]) // JSON numbers are float64
}

func TestAddField(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := zerolog.New(buf)
	event := logger.Info()

	// Test various field types
	event = addField(event, String("str", "value"))
	event = addField(event, Int("int", 42))
	event = addField(event, Int64("int64", int64(1000)))
	event = addField(event, Float64("float", 3.14))
	event = addField(event, Bool("bool", true))
	event = addField(event, Err(assert.AnError))
	event = addField(event, Duration("dur", time.Second))
	event = addField(event, Time("time", time.Unix(0, 0).UTC()))
	event = addField(event, Any("any", []string{"a", "b"}))

	event.Msg("test")

	output := buf.String()
	assert.Contains(t, output, "value")
	assert.Contains(t, output, "42")
	assert.Contains(t, output, "1000")
	assert.Contains(t, output, "3.14")
	assert.Contains(t, output, "true")
	assert.Contains(t, output, "assert.AnError")
}

func BenchmarkLogger(b *testing.B) {
	logger := NewLogger("info", "json", false)
	
	b.Run("Simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message")
		}
	})

	b.Run("WithFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message",
				String("key1", "value1"),
				Int("key2", 42),
				Float64("key3", 3.14),
			)
		}
	})

	b.Run("WithContext", func(b *testing.B) {
		ctx := context.WithValue(context.Background(), "trace_id", "abc123")
		for i := 0; i < b.N; i++ {
			logger.WithContext(ctx).Info("benchmark message")
		}
	})
}