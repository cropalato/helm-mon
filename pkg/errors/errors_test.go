package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelmError(t *testing.T) {
	baseErr := errors.New("base error")
	details := map[string]interface{}{
		"version": "1.0.0",
		"type":    "release",
	}

	tests := []struct {
		name     string
		err      error
		op       string
		chart    string
		details  map[string]interface{}
		wantStr  string
		wantBase error
	}{
		{
			name:     "basic helm error",
			err:      baseErr,
			op:       "install",
			chart:    "nginx",
			details:  details,
			wantStr:  `helm operation "install" failed for chart "nginx": base error (details: map[type:release version:1.0.0])`,
			wantBase: baseErr,
		},
		{
			name:     "helm error without chart",
			err:      baseErr,
			op:       "list",
			details:  nil,
			wantStr:  `helm operation "list" failed: base error`,
			wantBase: baseErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helmErr := NewHelmError(tt.op, tt.err, tt.chart, tt.details)

			assert.Equal(t, tt.wantStr, helmErr.Error())
			assert.True(t, errors.Is(helmErr, tt.wantBase))
			assert.True(t, IsHelmError(helmErr))
		})
	}
}

func TestRepositoryError(t *testing.T) {
	baseErr := errors.New("base error")

	tests := []struct {
		name     string
		err      error
		op       string
		repo     string
		wantStr  string
		wantBase error
	}{
		{
			name:     "basic repository error",
			err:      baseErr,
			op:       "sync",
			repo:     "stable",
			wantStr:  `repository operation "sync" failed for repo "stable": base error`,
			wantBase: baseErr,
		},
		{
			name:     "repository error without repo",
			err:      baseErr,
			op:       "list",
			wantStr:  `repository operation "list" failed: base error`,
			wantBase: baseErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoErr := NewRepositoryError(tt.op, tt.repo, tt.err)

			assert.Equal(t, tt.wantStr, repoErr.Error())
			assert.True(t, errors.Is(repoErr, tt.wantBase))
			assert.True(t, IsRepositoryError(repoErr))
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		value   interface{}
		message string
		want    string
	}{
		{
			name:    "validation error with value",
			field:   "version",
			value:   "invalid",
			message: "must be semver compliant",
			want:    `validation failed for field "version" with value invalid: must be semver compliant`,
		},
		{
			name:    "validation error without value",
			field:   "name",
			message: "required field",
			want:    `validation failed for field "name": required field`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validErr := NewValidationError(tt.field, tt.value, tt.message)

			assert.Equal(t, tt.want, validErr.Error())
			assert.True(t, IsValidationError(validErr))
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	baseErr := errors.New("base error")

	// Test error wrapping
	err := WrapHelmError(baseErr, "install", "nginx", nil)
	assert.True(t, errors.Is(err, baseErr))

	// Test error context
	contextErr := ErrorContextf(err, "failed to process chart %s", "nginx")
	assert.Contains(t, contextErr.Error(), "failed to process chart nginx")
	assert.True(t, errors.Is(contextErr, baseErr))
}

func TestErrorTypes(t *testing.T) {
	baseErr := errors.New("base error")

	tests := []struct {
		name  string
		err   error
		check func(error) bool
	}{
		{
			name:  "helm error",
			err:   NewHelmError("test", baseErr, "", nil),
			check: IsHelmError,
		},
		{
			name:  "repository error",
			err:   NewRepositoryError("test", "", baseErr),
			check: IsRepositoryError,
		},
		{
			name:  "metrics error",
			err:   NewMetricsError("test", "", baseErr),
			check: IsMetricsError,
		},
		{
			name:  "validation error",
			err:   NewValidationError("test", nil, "invalid"),
			check: IsValidationError,
		},
		{
			name:  "config error",
			err:   NewConfigError("test", nil, baseErr),
			check: IsConfigError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.check(tt.err))
		})
	}
}
