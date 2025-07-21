package errors

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		severity Severity
		category ErrorCategory
		retryable bool
	}{
		{
			code:      ErrCodeValidationFailed,
			severity:  SeverityLow,
			category:  CategoryUser,
			retryable: false,
		},
		{
			code:      ErrCodeFileNotFound,
			severity:  SeverityHigh,
			category:  CategorySystem,
			retryable: false,
		},
		{
			code:      ErrCodeConnectionTimeout,
			severity:  SeverityMedium,
			category:  CategoryExternal,
			retryable: true,
		},
		{
			code:      ErrCodeMemoryExhausted,
			severity:  SeverityCritical,
			category:  CategorySystem,
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			assert.Equal(t, tt.severity, GetSeverity(tt.code))
			assert.Equal(t, tt.category, GetCategory(tt.code))
			
			retryInfo := DefaultRetryInfo(tt.code)
			if tt.retryable {
				assert.NotEqual(t, RetryNever, retryInfo.Strategy)
			} else {
				assert.Equal(t, RetryNever, retryInfo.Strategy)
			}
		})
	}
}

func TestEnhancedError(t *testing.T) {
	t.Run("Basic error creation", func(t *testing.T) {
		err := NewEnhancedError(
			ErrCodeFileNotFound,
			"io",
			"readFile",
			"file not found: test.txt",
		)

		assert.NotNil(t, err)
		assert.Equal(t, ErrCodeFileNotFound, err.Code)
		assert.Equal(t, "io", err.Component)
		assert.Equal(t, "readFile", err.Operation)
		assert.Contains(t, err.Error(), "ERR_3000")
		assert.Equal(t, SeverityHigh, err.Severity)
		assert.Equal(t, CategorySystem, err.Category)
		assert.NotNil(t, err.RetryInfo)
		assert.False(t, err.IsRetryable())
		assert.NotEmpty(t, err.StackTrace)
		assert.NotNil(t, err.SystemInfo)
	})

	t.Run("Error with context", func(t *testing.T) {
		err := NewEnhancedError(
			ErrCodeProcessingFailed,
			"processor",
			"calculate",
			"calculation failed",
		)
		err.WithContext("input", "test_data")
		err.WithContext("iteration", 5)
		err.WithRequestID("req-123")
		err.WithUserMessage("Unable to process your request")
		err.WithSuggestion("Check input format")

		assert.Equal(t, "test_data", err.Context["input"])
		assert.Equal(t, 5, err.Context["iteration"])
		assert.Equal(t, "req-123", err.RequestID)
		assert.Equal(t, "Unable to process your request", err.UserMessage)
		assert.Contains(t, err.Suggestions, "Check input format")
	})

	t.Run("Wrapped error", func(t *testing.T) {
		originalErr := fmt.Errorf("original error")
		wrapped := WrapEnhanced(
			originalErr,
			ErrCodeIOTimeout,
			"network",
			"fetch",
			"timeout while fetching data",
		)

		assert.NotNil(t, wrapped)
		assert.Equal(t, originalErr, wrapped.Unwrap())
		assert.True(t, wrapped.IsRetryable())
		assert.Equal(t, RetryWithBackoff, wrapped.RetryInfo.Strategy)
	})

	t.Run("Error inheritance", func(t *testing.T) {
		parent := NewEnhancedError(
			ErrCodeProcessingFailed,
			"parent",
			"process",
			"parent error",
		)
		parent.WithContext("key1", "value1")
		parent.WithRequestID("req-123")
		parent.WithSuggestion("parent suggestion")

		child := WrapEnhanced(
			parent,
			ErrCodeCalculationFailed,
			"child",
			"calculate",
			"child error",
		)

		assert.Equal(t, "value1", child.Context["key1"])
		assert.Equal(t, "req-123", child.RequestID)
		assert.Contains(t, child.Suggestions, "parent suggestion")
	})

	t.Run("JSON serialization", func(t *testing.T) {
		err := NewEnhancedError(
			ErrCodeValidationFailed,
			"validator",
			"validate",
			"validation error",
		)
		err.WithContext("field", "email")

		jsonData, jsonErr := err.JSON()
		assert.NoError(t, jsonErr)
		assert.NotEmpty(t, jsonData)

		// Verify JSON structure
		var parsed map[string]interface{}
		assert.NoError(t, json.Unmarshal(jsonData, &parsed))
		assert.Equal(t, string(ErrCodeValidationFailed), parsed["Code"])
		assert.Equal(t, "validator", parsed["Component"])
	})
}

func TestSpecificErrorHelpers(t *testing.T) {
	t.Run("FileError", func(t *testing.T) {
		err := FileError(
			ErrCodeFileNotFound,
			"open",
			"/path/to/file.txt",
			fmt.Errorf("no such file"),
		)

		assert.Contains(t, err.Error(), "file operation failed")
		assert.Contains(t, err.Error(), "/path/to/file.txt")
		assert.Equal(t, "/path/to/file.txt", err.Context["filename"])
		assert.NotEmpty(t, err.Suggestions)
		assert.Contains(t, err.Suggestions[0], "Check if the file path is correct")
	})

	t.Run("NetworkError", func(t *testing.T) {
		err := NetworkError(
			ErrCodeConnectionTimeout,
			"GET",
			"https://api.example.com",
			fmt.Errorf("timeout"),
		)

		assert.Contains(t, err.Error(), "network operation failed")
		assert.Contains(t, err.Error(), "https://api.example.com")
		assert.Equal(t, "https://api.example.com", err.Context["url"])
		assert.True(t, err.IsRetryable())
		assert.NotEmpty(t, err.Suggestions)
	})

	t.Run("ValidationError", func(t *testing.T) {
		err := ValidationError("email", "invalid@", "valid email format")

		assert.Contains(t, err.Error(), "validation failed")
		assert.Equal(t, "email", err.Context["field"])
		assert.Equal(t, "invalid@", err.Context["value"])
		assert.Equal(t, "Invalid value for email", err.UserMessage)
		assert.False(t, err.IsRetryable())
	})
}

func TestRetryInfo(t *testing.T) {
	tests := []struct {
		code         ErrorCode
		expectRetry  bool
		strategy     RetryStrategy
		maxAttempts  int
	}{
		{
			code:        ErrCodeValidationFailed,
			expectRetry: false,
			strategy:    RetryNever,
		},
		{
			code:        ErrCodeFileReadFailed,
			expectRetry: true,
			strategy:    RetryWithBackoff,
			maxAttempts: 3,
		},
		{
			code:        ErrCodeConnectionTimeout,
			expectRetry: true,
			strategy:    RetryWithCircuitBreaker,
			maxAttempts: 5,
		},
		{
			code:        ErrCodeResourceBusy,
			expectRetry: true,
			strategy:    RetryWithBackoff,
			maxAttempts: 10,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			retryInfo := DefaultRetryInfo(tt.code)
			assert.Equal(t, tt.strategy, retryInfo.Strategy)
			
			if tt.expectRetry {
				assert.True(t, retryInfo.MaxAttempts > 0)
				assert.True(t, retryInfo.InitialDelay > 0)
				assert.True(t, retryInfo.MaxDelay > retryInfo.InitialDelay)
				assert.True(t, retryInfo.BackoffFactor > 1.0)
				
				if tt.maxAttempts > 0 {
					assert.Equal(t, tt.maxAttempts, retryInfo.MaxAttempts)
				}
			}
		})
	}
}

func TestErrorReporter(t *testing.T) {
	t.Run("Basic reporting", func(t *testing.T) {
		reporter := NewErrorReporter(100, time.Hour)

		// Report various errors
		err1 := NewEnhancedError(ErrCodeFileNotFound, "io", "read", "file not found")
		err2 := NewEnhancedError(ErrCodeValidationFailed, "validator", "check", "invalid input")
		err3 := NewEnhancedError(ErrCodeConnectionTimeout, "network", "fetch", "timeout")

		reporter.ReportError(err1)
		reporter.ReportError(err2)
		reporter.ReportError(err3)
		reporter.ReportError(err1) // Duplicate

		report := reporter.GetReport()

		assert.Equal(t, 4, report.TotalErrors)
		assert.Equal(t, 2, report.ErrorsByCode[ErrCodeFileNotFound])
		assert.Equal(t, 1, report.ErrorsByCode[ErrCodeValidationFailed])
		assert.Equal(t, 1, report.ErrorsByCode[ErrCodeConnectionTimeout])
		assert.Equal(t, 1, report.RetryableErrors) // Only connection timeout
		assert.NotEmpty(t, report.TopErrors)
		assert.NotEmpty(t, report.RecentErrors)
	})

	t.Run("System health calculation", func(t *testing.T) {
		reporter := NewErrorReporter(100, time.Hour)

		// Report errors of different severities
		for i := 0; i < 5; i++ {
			reporter.ReportError(NewEnhancedError(
				ErrCodeMemoryExhausted, "system", "allocate", "out of memory",
			))
		}
		for i := 0; i < 10; i++ {
			reporter.ReportError(NewEnhancedError(
				ErrCodeValidationFailed, "validator", "check", "invalid",
			))
		}

		report := reporter.GetReport()

		assert.Equal(t, 5, report.SystemHealth.CriticalErrors)
		assert.Less(t, report.SystemHealth.Score, 1.0)
		assert.NotEqual(t, "healthy", report.SystemHealth.Status)
		assert.NotEmpty(t, report.SystemHealth.Recommendations)
	})

	t.Run("Text report generation", func(t *testing.T) {
		reporter := NewErrorReporter(100, time.Hour)
		
		// Add some errors
		reporter.ReportError(FileError(
			ErrCodeFileNotFound, "open", "/test.txt", fmt.Errorf("not found"),
		))
		reporter.ReportError(NetworkError(
			ErrCodeConnectionTimeout, "GET", "https://api.test", fmt.Errorf("timeout"),
		))

		var buf strings.Builder
		err := reporter.WriteReport(&buf, "text")
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Error Report")
		assert.Contains(t, output, "Total Errors: 2")
		assert.Contains(t, output, "System Health:")
		assert.Contains(t, output, "Errors by Severity:")
	})

	t.Run("JSON report generation", func(t *testing.T) {
		reporter := NewErrorReporter(100, time.Hour)
		
		reporter.ReportError(NewEnhancedError(
			ErrCodeProcessingFailed, "processor", "process", "failed",
		))

		var buf strings.Builder
		err := reporter.WriteReport(&buf, "json")
		assert.NoError(t, err)

		var report ErrorReport
		assert.NoError(t, json.Unmarshal([]byte(buf.String()), &report))
		assert.Equal(t, 1, report.TotalErrors)
	})

	t.Run("Error rotation", func(t *testing.T) {
		reporter := NewErrorReporter(5, 0) // Max 5 errors, no time-based cleanup

		// Report more than max errors
		for i := 0; i < 10; i++ {
			reporter.ReportError(fmt.Errorf("error %d", i))
		}

		report := reporter.GetReport()
		assert.Equal(t, 5, report.TotalErrors)
	})
}

func TestErrorIs(t *testing.T) {
	err1 := NewEnhancedError(ErrCodeFileNotFound, "io", "read", "not found")
	err2 := NewEnhancedError(ErrCodeFileNotFound, "io", "write", "not found")
	err3 := NewEnhancedError(ErrCodeValidationFailed, "validator", "check", "invalid")

	assert.True(t, err1.Is(err2))  // Same code
	assert.False(t, err1.Is(err3)) // Different code

	// Test with ProcessorError
	procErr := &ProcessorError{
		Type: ErrTypeIO,
	}
	err1.Type = ErrTypeIO
	assert.True(t, err1.Is(procErr))
}

func BenchmarkErrorCreation(b *testing.B) {
	b.Run("NewEnhancedError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewEnhancedError(
				ErrCodeProcessingFailed,
				"benchmark",
				"test",
				"benchmark error",
			)
		}
	})

	b.Run("WrapEnhanced", func(b *testing.B) {
		baseErr := fmt.Errorf("base error")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = WrapEnhanced(
				baseErr,
				ErrCodeProcessingFailed,
				"benchmark",
				"test",
				"wrapped error",
			)
		}
	})
}

func BenchmarkErrorReporting(b *testing.B) {
	reporter := NewErrorReporter(10000, 0)
	err := NewEnhancedError(ErrCodeProcessingFailed, "bench", "test", "error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reporter.ReportError(err)
	}
}