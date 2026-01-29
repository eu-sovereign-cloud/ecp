package errors

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

// Sentinel errors for http handlers
var (
	ErrBadRequest         = errors.New("bad request")
	ErrPreconditionFailed = errors.New("precondition failed")
)

// DomainToSDKError converts a domain error to an RFC 7807 SDK error.
// This is the adapter's responsibility - mapping domain to protocol.
func DomainToSDKError(err error, requestPath string) schema.Error {
	// Check for domain Error with rich context
	if domainErr := model.AsError(err); domainErr != nil {
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
func convertDomainError(domainErr *model.Error, requestPath string) schema.Error {
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
func mapKindToHTTP(kind model.ErrKind) (int, string, schema.ErrorType) {
	switch kind {
	case model.KindForbidden:
		return http.StatusForbidden, "Forbidden", schema.ErrorTypeForbidden
	case model.KindNotFound:
		return http.StatusNotFound, "Not Found", schema.ErrorTypeResourceNotFound
	case model.KindConflict, model.KindAlreadyExists:
		return http.StatusConflict, "Conflict", schema.ErrorTypeResourceConflict
	case model.KindValidation:
		return http.StatusUnprocessableEntity, "Unprocessable Entity", schema.ErrorTypeValidationError
	case model.KindUnavailable:
		return http.StatusInternalServerError, "Service Unavailable", schema.ErrorTypeInternalServerError
	default:
		return http.StatusInternalServerError, "Internal Server Error", schema.ErrorTypeInternalServerError
	}
}

// mapErrorToHTTP maps non-domain errors to HTTP status, title, and type.
func mapErrorToHTTP(err error) (int, string, schema.ErrorType) {
	switch {
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, "Bad Request", schema.ErrorTypeInvalidRequest
	case errors.Is(err, ErrPreconditionFailed):
		return http.StatusPreconditionFailed, "Precondition Failed", schema.ErrorTypePreconditionFailed
	default:
		return http.StatusInternalServerError, "Internal Server Error", schema.ErrorTypeInternalServerError
	}
}

// WriteErrorResponse writes a structured error response according to RFC 7807.
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	sdkError := DomainToSDKError(err, r.URL.Path)

	logger.ErrorContext(r.Context(), "request error",
		slog.Int("status", int(sdkError.Status)),
		slog.String("type", sdkError.Type),
		slog.String("title", sdkError.Title),
		slog.Any("detail", sdkError.Detail),
		slog.Any("error", err),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(sdkError.Status))

	if encodeErr := json.NewEncoder(w).Encode(sdkError); encodeErr != nil {
		logger.ErrorContext(r.Context(), "failed to encode error response", slog.Any("error", encodeErr))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
