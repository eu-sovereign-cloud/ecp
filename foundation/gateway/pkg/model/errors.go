package model

import "errors"

// Sentinel errors for resource model operations.
var (
	// ErrForbidden - insufficient permissions to access the resource.
	ErrForbidden = errors.New("forbidden")

	// ErrNotFound - the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict - resource creation or modification conflict.
	ErrConflict = errors.New("conflict")

	// ErrValidation - resource validation failure.
	ErrValidation = errors.New("validation error")

	// ErrUnavailable - external service operation failure.
	ErrUnavailable = errors.New("service unavailable")

	// ErrAlreadyExists - resource already exists.
	ErrAlreadyExists = errors.New("resource already exists")
)

// ValidationError represents a domain validation error with details about what failed.
// This is a pure domain concept - field references are domain field names, not HTTP/JSON paths.
type ValidationError struct {
	// Message describes what validation failed
	Message string
	// Field is the domain field name that failed validation (e.g., "Size", "Name")
	Field string
	// Cause is the underlying error if any
	Cause error
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Message + " (field: " + e.Field + ")"
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// Is allows ValidationError to match ErrValidation
func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}
