package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// EnhancedError represents a comprehensive error with all metadata
type EnhancedError struct {
	// Core fields
	Code      ErrorCode              // Unique error code
	Type      ErrorType              // Error type (backward compatibility)
	Message   string                 // Human-readable message
	Component string                 // Component where error occurred
	Operation string                 // Operation being performed
	Cause     error                  // Underlying error

	// Enhanced fields
	Severity     Severity               // Error severity
	Category     ErrorCategory          // Error category
	RetryInfo    *RetryInfo             // Retry information
	Context      map[string]interface{} // Additional context
	Timestamp    time.Time              // When error occurred
	RequestID    string                 // Request/trace ID
	UserMessage  string                 // User-friendly message
	
	// Diagnostic info
	StackTrace   []StackFrame           // Stack trace
	SystemInfo   SystemInfo             // System information
	Suggestions  []string               // Suggested fixes
}

// StackFrame represents a single frame in the stack trace
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// SystemInfo contains system information at error time
type SystemInfo struct {
	Hostname    string
	GoVersion   string
	GOOS        string
	GOARCH      string
	NumCPU      int
	NumGoroutine int
	MemStats    *runtime.MemStats
}

// Error implements the error interface
func (e *EnhancedError) Error() string {
	if e.UserMessage != "" && e.Severity <= SeverityMedium {
		return e.UserMessage
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Component, e.Message)
}

// DetailedError returns a detailed error message
func (e *EnhancedError) DetailedError() string {
	details := fmt.Sprintf("[%s][%s][%s] %s: %s",
		e.Code, e.Type, e.Severity.String(), e.Component, e.Message)
	
	if e.Cause != nil {
		details += fmt.Sprintf(" (caused by: %v)", e.Cause)
	}
	
	if e.RequestID != "" {
		details += fmt.Sprintf(" [RequestID: %s]", e.RequestID)
	}
	
	return details
}

// JSON returns the error as JSON
func (e *EnhancedError) JSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// Unwrap returns the underlying error
func (e *EnhancedError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is interface
func (e *EnhancedError) Is(target error) bool {
	switch t := target.(type) {
	case *EnhancedError:
		return e.Code == t.Code || e.Type == t.Type
	case *ProcessorError:
		return e.Type == t.Type
	default:
		return false
	}
}

// IsRetryable checks if the error should be retried
func (e *EnhancedError) IsRetryable() bool {
	if e.RetryInfo == nil {
		return false
	}
	return e.RetryInfo.Strategy != RetryNever
}

// ShouldCircuitBreak checks if error should trigger circuit breaker
func (e *EnhancedError) ShouldCircuitBreak() bool {
	if e.RetryInfo == nil {
		return false
	}
	return e.RetryInfo.Strategy == RetryWithCircuitBreaker
}

// WithSuggestion adds a suggestion for fixing the error
func (e *EnhancedError) WithSuggestion(suggestion string) *EnhancedError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithRequestID adds a request ID to the error
func (e *EnhancedError) WithRequestID(requestID string) *EnhancedError {
	e.RequestID = requestID
	return e
}

// WithUserMessage sets a user-friendly message
func (e *EnhancedError) WithUserMessage(message string) *EnhancedError {
	e.UserMessage = message
	return e
}

// WithContext adds context to the error
func (e *EnhancedError) WithContext(key string, value interface{}) *EnhancedError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// NewEnhancedError creates a new enhanced error
func NewEnhancedError(code ErrorCode, component, operation, message string) *EnhancedError {
	pc := make([]uintptr, 10)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	
	var stackTrace []StackFrame
	for {
		frame, more := frames.Next()
		stackTrace = append(stackTrace, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}

	// Get system info
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	sysInfo := SystemInfo{
		GoVersion:    runtime.Version(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemStats:     &memStats,
	}

	return &EnhancedError{
		Code:       code,
		Type:       getTypeFromCode(code),
		Message:    message,
		Component:  component,
		Operation:  operation,
		Severity:   GetSeverity(code),
		Category:   GetCategory(code),
		RetryInfo:  DefaultRetryInfo(code),
		Context:    make(map[string]interface{}),
		Timestamp:  time.Now(),
		StackTrace: stackTrace,
		SystemInfo: sysInfo,
	}
}

// WrapEnhanced wraps an error with enhanced error information
func WrapEnhanced(err error, code ErrorCode, component, operation, message string) *EnhancedError {
	if err == nil {
		return nil
	}

	enhancedErr := NewEnhancedError(code, component, operation, message)
	enhancedErr.Cause = err

	// If wrapping another EnhancedError, inherit some fields
	if srcErr, ok := err.(*EnhancedError); ok {
		// Inherit context
		for k, v := range srcErr.Context {
			enhancedErr.Context[k] = v
		}
		// Inherit request ID if not set
		if enhancedErr.RequestID == "" {
			enhancedErr.RequestID = srcErr.RequestID
		}
		// Combine suggestions
		enhancedErr.Suggestions = append(enhancedErr.Suggestions, srcErr.Suggestions...)
	}

	return enhancedErr
}

// Helper functions for specific error types

// FileError creates a file-related error
func FileError(code ErrorCode, operation, filename string, err error) *EnhancedError {
	message := fmt.Sprintf("file operation failed on '%s'", filename)
	enhancedErr := WrapEnhanced(err, code, "filesystem", operation, message)
	enhancedErr.WithContext("filename", filename)
	
	// Add suggestions based on error code
	switch code {
	case ErrCodeFileNotFound:
		enhancedErr.WithSuggestion("Check if the file path is correct")
		enhancedErr.WithSuggestion("Ensure the file exists and is accessible")
	case ErrCodePermissionDenied:
		enhancedErr.WithSuggestion("Check file permissions")
		enhancedErr.WithSuggestion("Run with appropriate privileges")
	case ErrCodeDiskFull:
		enhancedErr.WithSuggestion("Free up disk space")
		enhancedErr.WithSuggestion("Check disk quota limits")
	}
	
	return enhancedErr
}

// NetworkError creates a network-related error
func NetworkError(code ErrorCode, operation, url string, err error) *EnhancedError {
	message := fmt.Sprintf("network operation failed for '%s'", url)
	enhancedErr := WrapEnhanced(err, code, "network", operation, message)
	enhancedErr.WithContext("url", url)
	
	switch code {
	case ErrCodeConnectionTimeout:
		enhancedErr.WithSuggestion("Check network connectivity")
		enhancedErr.WithSuggestion("Increase timeout duration")
	case ErrCodeDNSLookupFailed:
		enhancedErr.WithSuggestion("Check DNS configuration")
		enhancedErr.WithSuggestion("Verify the hostname is correct")
	}
	
	return enhancedErr
}

// ValidationErrorEnhanced creates a validation error
func ValidationErrorEnhanced(field, value, constraint string) *EnhancedError {
	message := fmt.Sprintf("validation failed for field '%s': value '%s' violates constraint '%s'", 
		field, value, constraint)
	enhancedErr := NewEnhancedError(ErrCodeValidationFailed, "validation", "validate", message)
	enhancedErr.WithContext("field", field)
	enhancedErr.WithContext("value", value)
	enhancedErr.WithContext("constraint", constraint)
	enhancedErr.WithUserMessage(fmt.Sprintf("Invalid value for %s", field))
	return enhancedErr
}

// ProcessingErrorEnhanced creates a processing error
func ProcessingErrorEnhanced(component, operation string, err error) *EnhancedError {
	message := fmt.Sprintf("processing failed in %s", operation)
	return WrapEnhanced(err, ErrCodeProcessingFailed, component, operation, message)
}

// Helper to convert error code to error type
func getTypeFromCode(code ErrorCode) ErrorType {
	switch code {
	case ErrCodeValidationFailed, ErrCodeInvalidFormat, ErrCodeMissingRequired:
		return ErrTypeValidation
	case ErrCodeFileNotFound, ErrCodeFileReadFailed, ErrCodeFileWriteFailed:
		return ErrTypeIO
	case ErrCodeProcessingFailed, ErrCodeParsingFailed, ErrCodeCalculationFailed:
		return ErrTypeProcessing
	case ErrCodeConfigNotFound, ErrCodeConfigInvalid:
		return ErrTypeConfiguration
	case ErrCodeTimeout, ErrCodeDeadlineExceeded:
		return ErrTypeTimeout
	case ErrCodeCanceled:
		return ErrTypeCanceled
	default:
		return ErrTypeUnknown
	}
}

// String methods for enums

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "LOW"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityHigh:
		return "HIGH"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

func (r RetryStrategy) String() string {
	switch r {
	case RetryNever:
		return "NEVER"
	case RetryImmediate:
		return "IMMEDIATE"
	case RetryWithBackoff:
		return "BACKOFF"
	case RetryWithCircuitBreaker:
		return "CIRCUIT_BREAKER"
	default:
		return "UNKNOWN"
	}
}