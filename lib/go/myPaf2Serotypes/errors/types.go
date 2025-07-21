// Package errors provides comprehensive error types and handling
package errors

// ErrorCode represents a unique error code for programmatic handling
type ErrorCode string

// Error codes for different error scenarios
const (
	// General errors (1000-1999)
	ErrCodeUnknown         ErrorCode = "ERR_1000"
	ErrCodeInternal        ErrorCode = "ERR_1001"
	ErrCodeInvalidArgument ErrorCode = "ERR_1002"
	ErrCodeNotImplemented  ErrorCode = "ERR_1003"

	// Validation errors (2000-2999)
	ErrCodeValidationFailed    ErrorCode = "ERR_2000"
	ErrCodeInvalidFormat       ErrorCode = "ERR_2001"
	ErrCodeMissingRequired     ErrorCode = "ERR_2002"
	ErrCodeOutOfRange          ErrorCode = "ERR_2003"
	ErrCodeInvalidType         ErrorCode = "ERR_2004"
	ErrCodeConstraintViolation ErrorCode = "ERR_2005"

	// I/O errors (3000-3999)
	ErrCodeFileNotFound     ErrorCode = "ERR_3000"
	ErrCodeFileReadFailed   ErrorCode = "ERR_3001"
	ErrCodeFileWriteFailed  ErrorCode = "ERR_3002"
	ErrCodeFileCreateFailed ErrorCode = "ERR_3003"
	ErrCodeFileCloseFailed  ErrorCode = "ERR_3004"
	ErrCodeDirCreateFailed  ErrorCode = "ERR_3005"
	ErrCodePermissionDenied ErrorCode = "ERR_3006"
	ErrCodeDiskFull         ErrorCode = "ERR_3007"
	ErrCodeIOTimeout        ErrorCode = "ERR_3008"

	// Network errors (4000-4999)
	ErrCodeNetworkUnreachable ErrorCode = "ERR_4000"
	ErrCodeConnectionRefused  ErrorCode = "ERR_4001"
	ErrCodeConnectionTimeout  ErrorCode = "ERR_4002"
	ErrCodeConnectionReset    ErrorCode = "ERR_4003"
	ErrCodeDNSLookupFailed    ErrorCode = "ERR_4004"
	ErrCodeSSLHandshakeFailed ErrorCode = "ERR_4005"

	// Processing errors (5000-5999)
	ErrCodeProcessingFailed   ErrorCode = "ERR_5000"
	ErrCodeParsingFailed      ErrorCode = "ERR_5001"
	ErrCodeCalculationFailed  ErrorCode = "ERR_5002"
	ErrCodeAssignmentFailed   ErrorCode = "ERR_5003"
	ErrCodePipelineFailed     ErrorCode = "ERR_5004"
	ErrCodeDataCorrupted      ErrorCode = "ERR_5005"
	ErrCodeIncompatibleData   ErrorCode = "ERR_5006"

	// Resource errors (6000-6999)
	ErrCodeResourceNotFound    ErrorCode = "ERR_6000"
	ErrCodeResourceBusy        ErrorCode = "ERR_6001"
	ErrCodeResourceExhausted   ErrorCode = "ERR_6002"
	ErrCodeMemoryExhausted     ErrorCode = "ERR_6003"
	ErrCodeQuotaExceeded       ErrorCode = "ERR_6004"
	ErrCodeRateLimitExceeded   ErrorCode = "ERR_6005"

	// Operation errors (7000-7999)
	ErrCodeTimeout           ErrorCode = "ERR_7000"
	ErrCodeCanceled          ErrorCode = "ERR_7001"
	ErrCodeDeadlineExceeded  ErrorCode = "ERR_7002"
	ErrCodeOperationAborted  ErrorCode = "ERR_7003"
	ErrCodeConcurrencyLimit  ErrorCode = "ERR_7004"

	// Configuration errors (8000-8999)
	ErrCodeConfigNotFound    ErrorCode = "ERR_8000"
	ErrCodeConfigInvalid     ErrorCode = "ERR_8001"
	ErrCodeConfigParseFailed ErrorCode = "ERR_8002"
	ErrCodeEnvVarMissing     ErrorCode = "ERR_8003"
)

// RetryStrategy defines when an error should be retried
type RetryStrategy int

const (
	// RetryNever indicates the error should never be retried
	RetryNever RetryStrategy = iota
	// RetryImmediate indicates immediate retry is safe
	RetryImmediate
	// RetryWithBackoff indicates retry with exponential backoff
	RetryWithBackoff
	// RetryWithCircuitBreaker indicates retry with circuit breaker
	RetryWithCircuitBreaker
)

// Severity represents the severity level of an error
type Severity int

const (
	// SeverityLow indicates a minor issue that doesn't affect functionality
	SeverityLow Severity = iota
	// SeverityMedium indicates an issue that affects some functionality
	SeverityMedium
	// SeverityHigh indicates a serious issue affecting core functionality
	SeverityHigh
	// SeverityCritical indicates a critical failure requiring immediate attention
	SeverityCritical
)

// ErrorCategory represents the category of error for routing and handling
type ErrorCategory string

const (
	CategoryUser          ErrorCategory = "user"          // User-caused errors
	CategorySystem        ErrorCategory = "system"        // System/infrastructure errors
	CategoryExternal      ErrorCategory = "external"      // External service errors
	CategoryConfiguration ErrorCategory = "configuration" // Configuration errors
	CategoryData          ErrorCategory = "data"          // Data-related errors
)

// RetryInfo contains information about retry strategy
type RetryInfo struct {
	Strategy        RetryStrategy
	MaxAttempts     int
	InitialDelay    int // milliseconds
	MaxDelay        int // milliseconds
	BackoffFactor   float64
	RetryableErrors []ErrorCode
}

// DefaultRetryInfo returns default retry configuration for an error code
func DefaultRetryInfo(code ErrorCode) *RetryInfo {
	switch code {
	// Never retry validation errors
	case ErrCodeValidationFailed, ErrCodeInvalidFormat, ErrCodeMissingRequired,
		ErrCodeOutOfRange, ErrCodeInvalidType, ErrCodeConstraintViolation:
		return &RetryInfo{Strategy: RetryNever}

	// Retry file operations with backoff
	case ErrCodeFileReadFailed, ErrCodeFileWriteFailed, ErrCodeFileCreateFailed:
		return &RetryInfo{
			Strategy:      RetryWithBackoff,
			MaxAttempts:   3,
			InitialDelay:  100,
			MaxDelay:      5000,
			BackoffFactor: 2.0,
		}

	// Retry network errors with circuit breaker
	case ErrCodeNetworkUnreachable, ErrCodeConnectionRefused, ErrCodeConnectionTimeout:
		return &RetryInfo{
			Strategy:      RetryWithCircuitBreaker,
			MaxAttempts:   5,
			InitialDelay:  500,
			MaxDelay:      30000,
			BackoffFactor: 2.0,
		}

	// Retry resource busy with backoff
	case ErrCodeResourceBusy, ErrCodeRateLimitExceeded:
		return &RetryInfo{
			Strategy:      RetryWithBackoff,
			MaxAttempts:   10,
			InitialDelay:  1000,
			MaxDelay:      60000,
			BackoffFactor: 1.5,
		}

	// Never retry configuration errors
	case ErrCodeConfigNotFound, ErrCodeConfigInvalid, ErrCodeConfigParseFailed:
		return &RetryInfo{Strategy: RetryNever}

	// Default: don't retry
	default:
		return &RetryInfo{Strategy: RetryNever}
	}
}

// GetSeverity returns the severity level for an error code
func GetSeverity(code ErrorCode) Severity {
	switch code {
	// Critical errors
	case ErrCodeInternal, ErrCodeMemoryExhausted, ErrCodeDiskFull,
		ErrCodeDataCorrupted, ErrCodeConfigNotFound:
		return SeverityCritical

	// High severity
	case ErrCodeFileNotFound, ErrCodePermissionDenied, ErrCodeProcessingFailed,
		ErrCodePipelineFailed, ErrCodeResourceExhausted:
		return SeverityHigh

	// Medium severity
	case ErrCodeConnectionTimeout, ErrCodeResourceBusy, ErrCodeRateLimitExceeded,
		ErrCodeTimeout, ErrCodeCanceled:
		return SeverityMedium

	// Low severity
	case ErrCodeValidationFailed, ErrCodeInvalidFormat:
		return SeverityLow

	default:
		return SeverityMedium
	}
}

// GetCategory returns the category for an error code
func GetCategory(code ErrorCode) ErrorCategory {
	switch code {
	case ErrCodeValidationFailed, ErrCodeInvalidFormat, ErrCodeMissingRequired,
		ErrCodeOutOfRange, ErrCodeInvalidType:
		return CategoryUser

	case ErrCodeFileNotFound, ErrCodeFileReadFailed, ErrCodeFileWriteFailed,
		ErrCodeMemoryExhausted, ErrCodeDiskFull:
		return CategorySystem

	case ErrCodeNetworkUnreachable, ErrCodeConnectionRefused, ErrCodeConnectionTimeout,
		ErrCodeDNSLookupFailed:
		return CategoryExternal

	case ErrCodeConfigNotFound, ErrCodeConfigInvalid, ErrCodeConfigParseFailed:
		return CategoryConfiguration

	case ErrCodeParsingFailed, ErrCodeDataCorrupted, ErrCodeIncompatibleData:
		return CategoryData

	default:
		return CategorySystem
	}
}