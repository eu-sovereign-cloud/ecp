package model

import "errors"

// ErrKind represents the category of domain error.
type ErrKind int

const (
	// KindForbidden indicates insufficient permissions to access the resource.
	KindForbidden ErrKind = iota
	// KindNotFound indicates the requested resource does not exist.
	KindNotFound
	// KindConflict indicates resource creation or modification conflict.
	KindConflict
	// KindValidation indicates resource validation failure.
	KindValidation
	// KindUnavailable indicates external service operation failure.
	KindUnavailable
	// KindAlreadyExists indicates resource already exists.
	KindAlreadyExists
)

// Sentinel errors
var (
	ErrForbidden     = NewError(KindForbidden, errors.New(KindForbidden.String()))
	ErrNotFound      = NewError(KindNotFound, errors.New(KindNotFound.String()))
	ErrConflict      = NewError(KindConflict, errors.New(KindConflict.String()))
	ErrValidation    = NewError(KindValidation, errors.New(KindValidation.String()))
	ErrUnavailable   = NewError(KindUnavailable, errors.New(KindUnavailable.String()))
	ErrAlreadyExists = NewError(KindAlreadyExists, errors.New(KindAlreadyExists.String()))
)

// String returns the string representation of the error kind.
func (k ErrKind) String() string {
	switch k {
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not found"
	case KindConflict:
		return "conflict"
	case KindValidation:
		return "validation error"
	case KindUnavailable:
		return "service unavailable"
	case KindAlreadyExists:
		return "resource already exists"
	default:
		return "unknown error"
	}
}

// ErrorSource represents the source of an error in domain terms.
type ErrorSource struct {
	// Name identifies the source of the error (e.g., field name, parameter name, resource name)
	Name string
	// Value is the value that caused the error, if applicable
	Value string
}

// Error represents a domain error with rich context.
type Error struct {
	// Kind categorizes the error type
	Kind ErrKind
	// Sources identifies which fields or parameters caused the error
	Sources []ErrorSource
	// Cause is the underlying error
	Cause error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Kind.String() + ": " + e.Cause.Error()
	}
	return e.Kind.String()
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithSource returns a new copy of the error with an additional source.
func (e *Error) WithSource(name, value string) *Error {
	sources := make([]ErrorSource, len(e.Sources), len(e.Sources)+1)
	copy(sources, e.Sources)
	sources = append(sources, ErrorSource{Name: name, Value: value})
	return &Error{
		Kind:    e.Kind,
		Sources: sources,
		Cause:   e.Cause,
	}
}

// NewError creates a new domain error with the given kind and cause.
func NewError(kind ErrKind, cause error, sources ...ErrorSource) *Error {
	return &Error{
		Kind:    kind,
		Cause:   cause,
		Sources: sources,
	}
}

// AsError attempts to extract a domain Error from err.
// Returns nil if err is not a domain Error.
func AsError(err error) *Error {
	var domainErr *Error
	if errors.As(err, &domainErr) {
		return domainErr
	}
	return nil
}

// Is allows Error to match by kind for errors.Is comparisons.
func (e *Error) Is(target error) bool {
	var te *Error
	if errors.As(target, &te) {
		return e.Kind == te.Kind
	}
	return false
}
