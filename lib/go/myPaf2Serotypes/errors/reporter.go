package errors

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"
)

// ErrorReport represents an aggregated error report
type ErrorReport struct {
	StartTime       time.Time                    `json:"start_time"`
	EndTime         time.Time                    `json:"end_time"`
	TotalErrors     int                          `json:"total_errors"`
	ErrorsByCode    map[ErrorCode]int            `json:"errors_by_code"`
	ErrorsByType    map[ErrorType]int            `json:"errors_by_type"`
	ErrorsBySeverity map[Severity]int            `json:"errors_by_severity"`
	ErrorsByComponent map[string]int              `json:"errors_by_component"`
	RecentErrors    []ErrorSummary               `json:"recent_errors"`
	TopErrors       []ErrorSummary               `json:"top_errors"`
	Suggestions     map[string]int               `json:"suggestions"`
	RetryableErrors int                          `json:"retryable_errors"`
	SystemHealth    SystemHealthStatus           `json:"system_health"`
}

// ErrorSummary represents a summary of an error for reporting
type ErrorSummary struct {
	Code       ErrorCode     `json:"code"`
	Message    string        `json:"message"`
	Component  string        `json:"component"`
	Operation  string        `json:"operation"`
	Count      int           `json:"count"`
	FirstSeen  time.Time     `json:"first_seen"`
	LastSeen   time.Time     `json:"last_seen"`
	Severity   Severity      `json:"severity"`
	IsRetryable bool         `json:"is_retryable"`
}

// SystemHealthStatus represents overall system health based on errors
type SystemHealthStatus struct {
	Status        string    `json:"status"` // "healthy", "degraded", "critical"
	Score         float64   `json:"score"`  // 0.0 to 1.0
	LastChecked   time.Time `json:"last_checked"`
	CriticalErrors int      `json:"critical_errors"`
	Recommendations []string `json:"recommendations"`
}

// ErrorReporter collects and reports on errors
type ErrorReporter struct {
	mu              sync.RWMutex
	errors          []*EnhancedError
	errorCounts     map[string]int // key: code+component+operation
	maxErrors       int
	retentionPeriod time.Duration
	startTime       time.Time
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(maxErrors int, retentionPeriod time.Duration) *ErrorReporter {
	return &ErrorReporter{
		errors:          make([]*EnhancedError, 0, maxErrors),
		errorCounts:     make(map[string]int),
		maxErrors:       maxErrors,
		retentionPeriod: retentionPeriod,
		startTime:       time.Now(),
	}
}

// ReportError adds an error to the reporter
func (r *ErrorReporter) ReportError(err error) {
	if err == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Convert to EnhancedError if needed
	var enhancedErr *EnhancedError
	switch e := err.(type) {
	case *EnhancedError:
		enhancedErr = e
	case *ProcessorError:
		enhancedErr = NewEnhancedError(
			ErrCodeProcessingFailed,
			e.Component,
			e.Operation,
			e.Message,
		)
		enhancedErr.Cause = e.Cause
	default:
		enhancedErr = NewEnhancedError(
			ErrCodeUnknown,
			"unknown",
			"unknown",
			err.Error(),
		)
	}

	// Add to errors list (with rotation)
	if len(r.errors) >= r.maxErrors {
		r.errors = r.errors[1:]
	}
	r.errors = append(r.errors, enhancedErr)

	// Update counts
	key := fmt.Sprintf("%s:%s:%s", enhancedErr.Code, enhancedErr.Component, enhancedErr.Operation)
	r.errorCounts[key]++

	// Clean up old errors
	r.cleanupOldErrors()
}

// GetReport generates an error report
func (r *ErrorReporter) GetReport() *ErrorReport {
	r.mu.RLock()
	defer r.mu.RUnlock()

	report := &ErrorReport{
		StartTime:         r.startTime,
		EndTime:           time.Now(),
		TotalErrors:       len(r.errors),
		ErrorsByCode:      make(map[ErrorCode]int),
		ErrorsByType:      make(map[ErrorType]int),
		ErrorsBySeverity:  make(map[Severity]int),
		ErrorsByComponent: make(map[string]int),
		RecentErrors:      make([]ErrorSummary, 0),
		TopErrors:         make([]ErrorSummary, 0),
		Suggestions:       make(map[string]int),
		RetryableErrors:   0,
	}

	// Aggregate error data
	errorSummaries := make(map[string]*ErrorSummary)
	
	for _, err := range r.errors {
		// Count by code, type, severity, component
		report.ErrorsByCode[err.Code]++
		report.ErrorsByType[err.Type]++
		report.ErrorsBySeverity[err.Severity]++
		report.ErrorsByComponent[err.Component]++

		// Count retryable errors
		if err.IsRetryable() {
			report.RetryableErrors++
		}

		// Aggregate suggestions
		for _, suggestion := range err.Suggestions {
			report.Suggestions[suggestion]++
		}

		// Create/update error summary
		key := fmt.Sprintf("%s:%s:%s", err.Code, err.Component, err.Operation)
		if summary, exists := errorSummaries[key]; exists {
			summary.Count++
			summary.LastSeen = err.Timestamp
		} else {
			errorSummaries[key] = &ErrorSummary{
				Code:        err.Code,
				Message:     err.Message,
				Component:   err.Component,
				Operation:   err.Operation,
				Count:       1,
				FirstSeen:   err.Timestamp,
				LastSeen:    err.Timestamp,
				Severity:    err.Severity,
				IsRetryable: err.IsRetryable(),
			}
		}
	}

	// Get recent errors (last 10)
	recentCount := 10
	if len(r.errors) < recentCount {
		recentCount = len(r.errors)
	}
	for i := len(r.errors) - recentCount; i < len(r.errors); i++ {
		err := r.errors[i]
		report.RecentErrors = append(report.RecentErrors, ErrorSummary{
			Code:        err.Code,
			Message:     err.Message,
			Component:   err.Component,
			Operation:   err.Operation,
			Count:       1,
			FirstSeen:   err.Timestamp,
			LastSeen:    err.Timestamp,
			Severity:    err.Severity,
			IsRetryable: err.IsRetryable(),
		})
	}

	// Get top errors by count
	var summaries []ErrorSummary
	for _, summary := range errorSummaries {
		summaries = append(summaries, *summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Count > summaries[j].Count
	})
	
	topCount := 10
	if len(summaries) < topCount {
		topCount = len(summaries)
	}
	report.TopErrors = summaries[:topCount]

	// Calculate system health
	report.SystemHealth = r.calculateSystemHealth(report)

	return report
}

// WriteReport writes the error report to a writer
func (r *ErrorReporter) WriteReport(w io.Writer, format string) error {
	report := r.GetReport()

	switch format {
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	
	case "text":
		fmt.Fprintf(w, "Error Report\n")
		fmt.Fprintf(w, "============\n\n")
		fmt.Fprintf(w, "Period: %s - %s\n", report.StartTime.Format(time.RFC3339), report.EndTime.Format(time.RFC3339))
		fmt.Fprintf(w, "Total Errors: %d\n", report.TotalErrors)
		fmt.Fprintf(w, "Retryable Errors: %d\n\n", report.RetryableErrors)

		fmt.Fprintf(w, "System Health: %s (Score: %.2f)\n", report.SystemHealth.Status, report.SystemHealth.Score)
		if len(report.SystemHealth.Recommendations) > 0 {
			fmt.Fprintf(w, "Recommendations:\n")
			for _, rec := range report.SystemHealth.Recommendations {
				fmt.Fprintf(w, "  - %s\n", rec)
			}
		}
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "Errors by Severity:\n")
		for severity := SeverityCritical; severity >= SeverityLow; severity-- {
			if count := report.ErrorsBySeverity[severity]; count > 0 {
				fmt.Fprintf(w, "  %s: %d\n", severity.String(), count)
			}
		}
		fmt.Fprintf(w, "\n")

		if len(report.TopErrors) > 0 {
			fmt.Fprintf(w, "Top Errors:\n")
			for i, err := range report.TopErrors {
				fmt.Fprintf(w, "  %d. [%s] %s (%d occurrences)\n", i+1, err.Code, err.Message, err.Count)
			}
			fmt.Fprintf(w, "\n")
		}

		if len(report.Suggestions) > 0 {
			fmt.Fprintf(w, "Top Suggestions:\n")
			// Sort suggestions by count
			var suggestions []struct {
				text  string
				count int
			}
			for text, count := range report.Suggestions {
				suggestions = append(suggestions, struct {
					text  string
					count int
				}{text, count})
			}
			sort.Slice(suggestions, func(i, j int) bool {
				return suggestions[i].count > suggestions[j].count
			})
			
			for i, sug := range suggestions {
				if i >= 5 {
					break
				}
				fmt.Fprintf(w, "  - %s (%d errors)\n", sug.text, sug.count)
			}
		}

		return nil

	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// Reset clears all error data
func (r *ErrorReporter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.errors = r.errors[:0]
	r.errorCounts = make(map[string]int)
	r.startTime = time.Now()
}

// cleanupOldErrors removes errors older than retention period
func (r *ErrorReporter) cleanupOldErrors() {
	if r.retentionPeriod == 0 {
		return
	}

	cutoff := time.Now().Add(-r.retentionPeriod)
	newErrors := make([]*EnhancedError, 0, len(r.errors))
	
	for _, err := range r.errors {
		if err.Timestamp.After(cutoff) {
			newErrors = append(newErrors, err)
		}
	}
	
	r.errors = newErrors
}

// calculateSystemHealth calculates system health based on errors
func (r *ErrorReporter) calculateSystemHealth(report *ErrorReport) SystemHealthStatus {
	health := SystemHealthStatus{
		Status:      "healthy",
		Score:       1.0,
		LastChecked: time.Now(),
		CriticalErrors: report.ErrorsBySeverity[SeverityCritical],
		Recommendations: make([]string, 0),
	}

	// Deduct points based on error severity
	criticalPenalty := float64(report.ErrorsBySeverity[SeverityCritical]) * 0.3
	highPenalty := float64(report.ErrorsBySeverity[SeverityHigh]) * 0.15
	mediumPenalty := float64(report.ErrorsBySeverity[SeverityMedium]) * 0.05
	lowPenalty := float64(report.ErrorsBySeverity[SeverityLow]) * 0.01

	health.Score -= criticalPenalty + highPenalty + mediumPenalty + lowPenalty
	if health.Score < 0 {
		health.Score = 0
	}

	// Determine status
	switch {
	case health.Score < 0.5:
		health.Status = "critical"
	case health.Score < 0.8:
		health.Status = "degraded"
	default:
		health.Status = "healthy"
	}

	// Add recommendations
	if report.ErrorsBySeverity[SeverityCritical] > 0 {
		health.Recommendations = append(health.Recommendations, 
			"Critical errors detected - immediate attention required")
	}
	
	if report.RetryableErrors > report.TotalErrors/2 {
		health.Recommendations = append(health.Recommendations,
			"High number of retryable errors - check system resources and connectivity")
	}

	// Add top suggestions
	for suggestion, count := range report.Suggestions {
		if count > 5 {
			health.Recommendations = append(health.Recommendations, suggestion)
		}
		if len(health.Recommendations) >= 5 {
			break
		}
	}

	return health
}

// GlobalReporter is a singleton error reporter
var (
	globalReporter *ErrorReporter
	globalMu       sync.Once
)

// GetGlobalReporter returns the global error reporter
func GetGlobalReporter() *ErrorReporter {
	globalMu.Do(func() {
		globalReporter = NewErrorReporter(10000, 24*time.Hour)
	})
	return globalReporter
}

// ReportGlobalError reports an error to the global reporter
func ReportGlobalError(err error) {
	GetGlobalReporter().ReportError(err)
}