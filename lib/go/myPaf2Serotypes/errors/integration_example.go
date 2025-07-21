package errors

import (
	"context"
	"fmt"
)

// Example integration patterns for the enhanced error system

// ExampleFileOperation shows how to use enhanced errors for file operations
func ExampleFileOperation(filename string) error {
	// Simulate file operation
	err := fmt.Errorf("permission denied")
	
	// Create enhanced error with all metadata
	return FileError(
		ErrCodePermissionDenied,
		"open",
		filename,
		err,
	).WithRequestID("req-123").
		WithContext("user", "john").
		WithContext("mode", "read-only").
		WithUserMessage("Unable to access the requested file. Please check your permissions.")
}

// ExampleNetworkOperation shows network error handling
func ExampleNetworkOperation(ctx context.Context, url string) error {
	// Check for context cancellation
	if ctx.Err() != nil {
		return NewEnhancedError(
			ErrCodeCanceled,
			"network",
			"fetch",
			"operation canceled by user",
		).WithContext("url", url)
	}

	// Simulate network timeout
	err := fmt.Errorf("connection timeout after 30s")
	
	return NetworkError(
		ErrCodeConnectionTimeout,
		"GET",
		url,
		err,
	).WithContext("timeout", "30s").
		WithContext("retry_count", 3)
}

// ExampleValidation shows validation error handling
func ExampleValidation(data map[string]interface{}) error {
	// Check required fields
	email, ok := data["email"].(string)
	if !ok || email == "" {
		return ValidationErrorEnhanced(
			"email",
			fmt.Sprintf("%v", data["email"]),
			"required field",
		).WithUserMessage("Email address is required")
	}

	// Validate format
	if !isValidEmail(email) {
		return ValidationErrorEnhanced(
			"email",
			email,
			"valid email format",
		).WithSuggestion("Email should be in format: user@example.com")
	}

	return nil
}

// ExamplePipelineError shows how to aggregate errors in a pipeline
func ExamplePipelineError(ctx context.Context) error {
	errorList := NewErrorList()

	// Stage 1: Read input
	if err := readInput(); err != nil {
		errorList.Add(WrapEnhanced(
			err,
			ErrCodeFileReadFailed,
			"pipeline",
			"readInput",
			"failed to read input file",
		))
	}

	// Stage 2: Process data
	if err := processData(); err != nil {
		errorList.Add(WrapEnhanced(
			err,
			ErrCodeProcessingFailed,
			"pipeline",
			"processData",
			"data processing failed",
		))
	}

	// Stage 3: Write output
	if err := writeOutput(); err != nil {
		errorList.Add(WrapEnhanced(
			err,
			ErrCodeFileWriteFailed,
			"pipeline",
			"writeOutput",
			"failed to write output",
		))
	}

	if errorList.HasErrors() {
		// Create a pipeline error with all sub-errors
		pipelineErr := NewEnhancedError(
			ErrCodePipelineFailed,
			"pipeline",
			"execute",
			fmt.Sprintf("pipeline failed with %d errors", len(errorList.Errors)),
		)
		
		// Add each error to context
		for i, err := range errorList.Errors {
			pipelineErr.WithContext(fmt.Sprintf("error_%d", i), err.Error())
		}
		
		return pipelineErr
	}

	return nil
}

// ExampleErrorReporting shows how to use the error reporter
func ExampleErrorReporting() {
	// Get global reporter
	reporter := GetGlobalReporter()

	// Simulate various operations with error reporting
	for i := 0; i < 10; i++ {
		if err := simulateOperation(i); err != nil {
			reporter.ReportError(err)
		}
	}

	// Generate report
	report := reporter.GetReport()
	
	// Check system health
	if report.SystemHealth.Status == "critical" {
		fmt.Printf("System in critical state: %.2f health score\n", report.SystemHealth.Score)
		for _, rec := range report.SystemHealth.Recommendations {
			fmt.Printf("- %s\n", rec)
		}
	}
}

// ExampleRetryableError shows how to check if an error should be retried
func ExampleRetryableError(operation func() error) error {
	var lastErr error
	maxAttempts := 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Check if error is enhanced and retryable
		if enhancedErr, ok := err.(*EnhancedError); ok {
			if !enhancedErr.IsRetryable() {
				return err // Don't retry
			}

			// Get retry info
			retryInfo := enhancedErr.RetryInfo
			if attempt < retryInfo.MaxAttempts {
				// Calculate delay with exponential backoff
				delay := calculateBackoff(
					retryInfo.InitialDelay,
					retryInfo.BackoffFactor,
					attempt,
					retryInfo.MaxDelay,
				)
				
				// Log retry attempt
				fmt.Printf("Retrying operation (attempt %d/%d) after %dms\n", 
					attempt, retryInfo.MaxAttempts, delay)
				
				// Sleep before retry
				// time.Sleep(time.Duration(delay) * time.Millisecond)
				
				lastErr = enhancedErr.WithContext("retry_attempt", attempt)
				continue
			}
		}

		return err
	}

	return WrapEnhanced(
		lastErr,
		ErrCodeOperationAborted,
		"retry",
		"execute",
		fmt.Sprintf("operation failed after %d attempts", maxAttempts),
	)
}

// Helper functions for examples

func isValidEmail(email string) bool {
	// Simplified email validation
	return len(email) > 3 && len(email) < 255
}

func readInput() error {
	// Simulate operation
	return nil
}

func processData() error {
	// Simulate operation
	return nil
}

func writeOutput() error {
	// Simulate operation
	return nil
}

func simulateOperation(i int) error {
	switch i % 5 {
	case 0:
		return FileError(ErrCodeFileNotFound, "open", fmt.Sprintf("file%d.txt", i), fmt.Errorf("not found"))
	case 1:
		return ValidationErrorEnhanced("field", fmt.Sprintf("value%d", i), "positive number")
	case 2:
		return NetworkError(ErrCodeConnectionTimeout, "GET", fmt.Sprintf("https://api.example.com/%d", i), fmt.Errorf("timeout"))
	case 3:
		return nil // Success
	default:
		return ProcessingErrorEnhanced("processor", fmt.Sprintf("task%d", i), fmt.Errorf("unknown error"))
	}
}

func calculateBackoff(initialDelay int, factor float64, attempt int, maxDelay int) int {
	delay := float64(initialDelay) * pow(factor, float64(attempt-1))
	if int(delay) > maxDelay {
		return maxDelay
	}
	return int(delay)
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}