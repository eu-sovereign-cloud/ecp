package model

import "errors"

// Sentinel errors for resource model operations that mirror HTTP status codes.
var (
	// ErrForbidden - 403
	ErrForbidden = errors.New("forbidden")

	// ErrNotFound - 404
	ErrNotFound = errors.New("not found")

	// ErrConflict - 409
	ErrConflict = errors.New("conflict")

	// ErrValidation - 422
	ErrValidation = errors.New("validation error")

	// ErrUnavailable - 500
	ErrUnavailable = errors.New("service unavailable")
)
