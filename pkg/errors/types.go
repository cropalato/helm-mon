package errors

import (
	"fmt"
	"strings"
)

// Common error types
type (
	// HelmError wraps helm-related errors with context
	HelmError struct {
		Op      string                 // Operation that failed
		Err     error                  // Original error
		Chart   string                 // Chart name if applicable
		Details map[string]interface{} // Additional context
	}

	// RepositoryError wraps repository-related errors
	RepositoryError struct {
		Op   string // Operation that failed
		Repo string // Repository name or path
		Err  error  // Original error
	}

	// MetricsError wraps metrics-related errors
	MetricsError struct {
		Op         string // Operation that failed
		Err        error  // Original error
		MetricName string // Name of the metric
	}

	// ValidationError wraps validation-related errors
	ValidationError struct {
		Field   string      // Field that failed validation
		Value   interface{} // Invalid value
		Message string      // Validation message
	}

	// ConfigError wraps configuration-related errors
	ConfigError struct {
		Parameter string      // Parameter that caused the error
		Value     interface{} // Invalid value
		Err       error       // Original error
	}
)

// Error implementations

func (e *HelmError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "helm operation %q failed", e.Op)
	if e.Chart != "" {
		fmt.Fprintf(&sb, " for chart %q", e.Chart)
	}
	if e.Err != nil {
		fmt.Fprintf(&sb, ": %v", e.Err)
	}
	if len(e.Details) > 0 {
		fmt.Fprintf(&sb, " (details: %v)", e.Details)
	}
	return sb.String()
}

func (e *HelmError) Unwrap() error {
	return e.Err
}

func (e *RepositoryError) Error() string {
	if e.Repo != "" {
		return fmt.Sprintf("repository operation %q failed for repo %q: %v", e.Op, e.Repo, e.Err)
	}
	return fmt.Sprintf("repository operation %q failed: %v", e.Op, e.Err)
}

func (e *RepositoryError) Unwrap() error {
	return e.Err
}

func (e *MetricsError) Error() string {
	if e.MetricName != "" {
		return fmt.Sprintf("metrics operation %q failed for metric %q: %v", e.Op, e.MetricName, e.Err)
	}
	return fmt.Sprintf("metrics operation %q failed: %v", e.Op, e.Err)
}

func (e *MetricsError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) Error() string {
	if e.Value != nil {
		return fmt.Sprintf("validation failed for field %q with value %v: %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed for field %q: %s", e.Field, e.Message)
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error for parameter %q with value %v: %v", e.Parameter, e.Value, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
