// Package errors provides custom error types and error handling utilities
package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorType represents the category of error
type ErrorType int

const (
	// ErrTypeUnknown represents an unknown error type
	ErrTypeUnknown ErrorType = iota
	// ErrTypeValidation represents a validation error
	ErrTypeValidation
	// ErrTypeIO represents an I/O error
	ErrTypeIO
	// ErrTypeProcessing represents a processing error
	ErrTypeProcessing
	// ErrTypeConfiguration represents a configuration error
	ErrTypeConfiguration
	// ErrTypeNotFound represents a not found error
	ErrTypeNotFound
	// ErrTypeTimeout represents a timeout error
	ErrTypeTimeout
	// ErrTypeCanceled represents a canceled operation error
	ErrTypeCanceled
)

// String returns the string representation of the error type
func (t ErrorType) String() string {
	switch t {
	case ErrTypeValidation:
		return "validation"
	case ErrTypeIO:
		return "io"
	case ErrTypeProcessing:
		return "processing"
	case ErrTypeConfiguration:
		return "configuration"
	case ErrTypeNotFound:
		return "not_found"
	case ErrTypeTimeout:
		return "timeout"
	case ErrTypeCanceled:
		return "canceled"
	default:
		return "unknown"
	}
}

// ProcessorError represents a comprehensive error with context
type ProcessorError struct {
	Type      ErrorType              // Error category
	Component string                 // Component where error occurred
	Operation string                 // Operation being performed
	Message   string                 // Human-readable error message
	Cause     error                  // Underlying error
	Context   map[string]interface{} // Additional context
	Timestamp time.Time              // When the error occurred
	File      string                 // Source file
	Line      int                    // Line number
}

// Error implements the error interface
func (e *ProcessorError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Type, e.Component, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Component, e.Message)
}

// Unwrap returns the underlying error
func (e *ProcessorError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is interface
func (e *ProcessorError) Is(target error) bool {
	t, ok := target.(*ProcessorError)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// WithContext adds context to the error
func (e *ProcessorError) WithContext(key string, value interface{}) *ProcessorError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new ProcessorError
func New(errType ErrorType, component, operation, message string) *ProcessorError {
	_, file, line, _ := runtime.Caller(1)
	return &ProcessorError{
		Type:      errType,
		Component: component,
		Operation: operation,
		Message:   message,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Context:   make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with ProcessorError
func Wrap(err error, errType ErrorType, component, operation, message string) *ProcessorError {
	if err == nil {
		return nil
	}
	
	_, file, line, _ := runtime.Caller(1)
	pe := &ProcessorError{
		Type:      errType,
		Component: component,
		Operation: operation,
		Message:   message,
		Cause:     err,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Context:   make(map[string]interface{}),
	}

	// If wrapping another ProcessorError, inherit its context
	if srcErr, ok := err.(*ProcessorError); ok {
		for k, v := range srcErr.Context {
			pe.Context[k] = v
		}
	}

	return pe
}

// Common error constructors

// ValidationError creates a validation error
func ValidationError(component, operation, message string) *ProcessorError {
	return New(ErrTypeValidation, component, operation, message)
}

// IOError creates an I/O error
func IOError(component, operation, message string, err error) *ProcessorError {
	return Wrap(err, ErrTypeIO, component, operation, message)
}

// ProcessingError creates a processing error
func ProcessingError(component, operation, message string) *ProcessorError {
	return New(ErrTypeProcessing, component, operation, message)
}

// ConfigError creates a configuration error
func ConfigError(message string) *ProcessorError {
	return New(ErrTypeConfiguration, "config", "validate", message)
}

// NotFoundError creates a not found error
func NotFoundError(component, operation, resource string) *ProcessorError {
	return New(ErrTypeNotFound, component, operation, fmt.Sprintf("%s not found", resource))
}

// TimeoutError creates a timeout error
func TimeoutError(component, operation string, duration time.Duration) *ProcessorError {
	err := New(ErrTypeTimeout, component, operation, fmt.Sprintf("operation timed out after %v", duration))
	err.WithContext("timeout_duration", duration)
	return err
}

// CanceledError creates a canceled operation error
func CanceledError(component, operation string) *ProcessorError {
	return New(ErrTypeCanceled, component, operation, "operation was canceled")
}

// ErrorList represents a collection of errors
type ErrorList struct {
	Errors []error
}

// Error implements the error interface
func (e *ErrorList) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d errors occurred", len(e.Errors))
}

// Add adds an error to the list
func (e *ErrorList) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (e *ErrorList) HasErrors() bool {
	return len(e.Errors) > 0
}

// First returns the first error or nil
func (e *ErrorList) First() error {
	if len(e.Errors) > 0 {
		return e.Errors[0]
	}
	return nil
}

// All returns all errors
func (e *ErrorList) All() []error {
	return e.Errors
}

// NewErrorList creates a new error list
func NewErrorList() *ErrorList {
	return &ErrorList{
		Errors: make([]error, 0),
	}
}

// IsType checks if an error is of a specific type
func IsType(err error, errType ErrorType) bool {
	if err == nil {
		return false
	}
	if pe, ok := err.(*ProcessorError); ok {
		return pe.Type == errType
	}
	return false
}

// GetType returns the error type or ErrTypeUnknown
func GetType(err error) ErrorType {
	if err == nil {
		return ErrTypeUnknown
	}
	if pe, ok := err.(*ProcessorError); ok {
		return pe.Type
	}
	return ErrTypeUnknown
}