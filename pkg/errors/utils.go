package errors

import (
	"errors"
	"fmt"
)

// Common error checks
var (
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
)

// IsHelmError checks if the error is a HelmError
func IsHelmError(err error) bool {
	var helmErr *HelmError
	return As(err, &helmErr)
}

// IsRepositoryError checks if the error is a RepositoryError
func IsRepositoryError(err error) bool {
	var repoErr *RepositoryError
	return As(err, &repoErr)
}

// IsMetricsError checks if the error is a MetricsError
func IsMetricsError(err error) bool {
	var metricsErr *MetricsError
	return As(err, &metricsErr)
}

// IsValidationError checks if the error is a ValidationError
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return As(err, &validationErr)
}

// IsConfigError checks if the error is a ConfigError
func IsConfigError(err error) bool {
	var configErr *ConfigError
	return As(err, &configErr)
}

// Error creation helpers

// NewHelmError creates a new HelmError with the given details
func NewHelmError(op string, err error, chart string, details map[string]interface{}) error {
	return &HelmError{
		Op:      op,
		Err:     err,
		Chart:   chart,
		Details: details,
	}
}

// NewRepositoryError creates a new RepositoryError
func NewRepositoryError(op string, repo string, err error) error {
	return &RepositoryError{
		Op:   op,
		Repo: repo,
		Err:  err,
	}
}

// NewMetricsError creates a new MetricsError
func NewMetricsError(op string, metricName string, err error) error {
	return &MetricsError{
		Op:         op,
		MetricName: metricName,
		Err:        err,
	}
}

// NewValidationError creates a new ValidationError
func NewValidationError(field string, value interface{}, message string) error {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewConfigError creates a new ConfigError
func NewConfigError(parameter string, value interface{}, err error) error {
	return &ConfigError{
		Parameter: parameter,
		Value:     value,
		Err:       err,
	}
}

// Error wrapping helpers

// WrapHelmError wraps an existing error with helm context
func WrapHelmError(err error, op string, chart string, details map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return &HelmError{
		Op:      op,
		Err:     err,
		Chart:   chart,
		Details: details,
	}
}

// WrapRepositoryError wraps an existing error with repository context
func WrapRepositoryError(err error, op string, repo string) error {
	if err == nil {
		return nil
	}
	return &RepositoryError{
		Op:   op,
		Repo: repo,
		Err:  err,
	}
}

// ErrorContextf adds context to an error
func ErrorContextf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}
