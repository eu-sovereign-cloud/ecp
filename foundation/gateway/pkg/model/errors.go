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

	ErrAlreadyExists = errors.New("resource already exists")
)
