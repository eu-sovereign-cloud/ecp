package httperror

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

// ErrBadRequest is returned when the request body cannot be decoded.
var ErrBadRequest = errors.New("bad request")

// DomainToAPIError converts a domain error to an RFC 7807 SDK error.
// This is the adapter's responsibility — mapping domain to protocol.
func DomainToAPIError(err error, requestPath string) schema.Error {
	// Check for domain Error with rich context.
	if domainErr := kernel.AsError(err); domainErr != nil {
		return convertDomainError(domainErr, requestPath)
	}

	// Map non-domain errors to HTTP responses.
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

// convertDomainError converts a kernel.Error to an SDK error with full context.
func convertDomainError(domainErr *kernel.Error, requestPath string) schema.Error {
	status, title, errorType := mapKindToHTTP(domainErr.Kind)

	sdkErr := schema.Error{
		Type:     string(errorType),
		Title:    title,
		Status:   float32(status),
		Instance: requestPath,
		Detail:   domainErr.Error(),
	}

	// Convert domain sources to SDK sources.
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

// mapKindToHTTP maps kernel error kinds to HTTP status, title, and RFC 7807 type.
func mapKindToHTTP(kind kernel.ErrKind) (int, string, schema.ErrorType) {
	switch kind {
	case kernel.KindForbidden:
		return http.StatusForbidden, kernel.KindForbidden.String(), schema.ErrorTypeForbidden
	case kernel.KindNotFound:
		return http.StatusNotFound, kernel.KindNotFound.String(), schema.ErrorTypeResourceNotFound
	case kernel.KindConflict, kernel.KindAlreadyExists:
		return http.StatusConflict, kernel.KindConflict.String(), schema.ErrorTypeResourceConflict
	case kernel.KindPreconditionFailed:
		return http.StatusPreconditionFailed, kernel.KindPreconditionFailed.String(), schema.ErrorTypePreconditionFailed
	case kernel.KindValidation:
		return http.StatusUnprocessableEntity, kernel.KindValidation.String(), schema.ErrorTypeValidationError
	case kernel.KindUnavailable:
		return http.StatusInternalServerError, kernel.KindUnavailable.String(), schema.ErrorTypeInternalServerError
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), schema.ErrorTypeInternalServerError
	}
}

// mapErrorToHTTP maps non-domain errors to HTTP status, title, and RFC 7807 type.
func mapErrorToHTTP(err error) (int, string, schema.ErrorType) {
	switch {
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, http.StatusText(http.StatusBadRequest), schema.ErrorTypeInvalidRequest
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), schema.ErrorTypeInternalServerError
	}
}
