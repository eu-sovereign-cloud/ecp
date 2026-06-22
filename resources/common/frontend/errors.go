package frontend

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

var (
	ErrBadRequest = errors.New("bad request")
)

// ErrKind represents the category of domain error.
type ErrKind int

const (
	// KindForbidden indicates insufficient permissions to access the resource.
	KindForbidden ErrKind = iota
	// KindNotFound indicates the requested resource does not exist.
	KindNotFound
	// KindConflict indicates resource creation or modification conflict.
	KindConflict
	// KindPreconditionFailed indicates resource modification/deletion failed due to resourceVersion issues
	KindPreconditionFailed
	// KindValidation indicates resource validation failure.
	KindValidation
	// KindUnavailable indicates external service operation failure.
	KindUnavailable
	// KindAlreadyExists indicates resource already exists.
	KindAlreadyExists
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
	case KindPreconditionFailed:
		return "precondition failed"
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

// AsError attempts to extract a domain Error from err.
// Returns nil if err is not a domain Error.
func AsError(err error) *Error {
	var domainErr *Error
	if errors.As(err, &domainErr) {
		return domainErr
	}
	return nil
}

// DomainToAPIError converts a domain error to an RFC 7807 SDK error.
// This is the adapter's responsibility - mapping domain to protocol.
func DomainToAPIError(err error, requestPath string) schema.Error {
	// Check for domain Error with rich context
	if domainErr := AsError(err); domainErr != nil {
		return convertDomainError(domainErr, requestPath)
	}

	// Map non-domain errors to HTTP responses
	status, title, errorType := mapErrorToHTTP(err)

	sdkErr := schema.Error{
		Type:     string(errorType),
		Title:    title,
		Status:   float32(status),
		Instance: requestPath,
	}

	detail := err.Error()
	if detail != "" {
		sdkErr.Detail = detail
	}

	return sdkErr
}

// convertDomainError converts a domain Error to SDK error with full context.
func convertDomainError(domainErr *Error, requestPath string) schema.Error {
	status, title, errorType := mapKindToHTTP(domainErr.Kind)

	sdkErr := schema.Error{
		Type:     string(errorType),
		Title:    title,
		Status:   float32(status),
		Instance: requestPath,
		Detail:   domainErr.Error(),
	}

	// Convert domain sources to SDK sources
	if len(domainErr.Sources) > 0 {
		sdkErr.Sources = make([]schema.ErrorSource, len(domainErr.Sources))
		for i, src := range domainErr.Sources {
			sdkErr.Sources[i] = schema.ErrorSource{
				Pointer:   src.Name,
				Parameter: src.Value,
			}
		}
	}

	return sdkErr
}

// mapKindToHTTP maps domain error kinds to HTTP status, title, and type.
func mapKindToHTTP(kind ErrKind) (int, string, schema.ErrorType) {
	switch kind {
	case KindForbidden:
		return http.StatusForbidden, KindForbidden.String(), schema.ErrorTypeForbidden
	case KindNotFound:
		return http.StatusNotFound, KindNotFound.String(), schema.ErrorTypeResourceNotFound
	case KindConflict, KindAlreadyExists:
		return http.StatusConflict, KindConflict.String(), schema.ErrorTypeResourceConflict
	case KindPreconditionFailed:
		return http.StatusPreconditionFailed, KindPreconditionFailed.String(), schema.ErrorTypePreconditionFailed
	case KindValidation:
		return http.StatusUnprocessableEntity, KindValidation.String(), schema.ErrorTypeValidationError
	case KindUnavailable:
		return http.StatusInternalServerError, KindUnavailable.String(), schema.ErrorTypeInternalServerError
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), schema.ErrorTypeInternalServerError
	}
}

// mapErrorToHTTP maps non-domain errors to HTTP status, title, and type.
func mapErrorToHTTP(err error) (int, string, schema.ErrorType) {
	switch {
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, http.StatusText(http.StatusBadRequest), schema.ErrorTypeInvalidRequest
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), schema.ErrorTypeInternalServerError
	}
}

// WriteErrorResponse writes a structured error response according to RFC 7807.
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	sdkError := DomainToAPIError(err, r.URL.Path)

	logger.ErrorContext(r.Context(), "request error",
		slog.Int("status", int(sdkError.Status)),
		slog.String("type", sdkError.Type),
		slog.String("title", sdkError.Title),
		slog.Any("detail", sdkError.Detail),
		slog.Any("error", err),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(sdkError.Status))

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if encodeErr := enc.Encode(sdkError); encodeErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode error response", slog.Any("error", encodeErr))
	}
	_, _ = w.Write(buf.Bytes())
}
