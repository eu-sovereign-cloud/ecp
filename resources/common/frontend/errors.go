package frontend

import (
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
)

// ErrBadRequest is returned when the request body cannot be decoded.
// Re-exported from framework/frontend/rest so existing call sites compile unchanged.
var ErrBadRequest = rest.ErrBadRequest

// WriteErrorResponse writes a structured error response according to RFC 7807.
// Delegates to framework/frontend/rest.WriteErrorResponse.
func WriteErrorResponse(w http.ResponseWriter, r *http.Request, l *slog.Logger, err error) {
	rest.WriteErrorResponse(w, r, l, err)
}

// DomainToAPIError converts a domain error to an RFC 7807 SDK error.
// Delegates to framework/frontend/rest.DomainToAPIError.
func DomainToAPIError(err error, p string) schema.Error {
	return rest.DomainToAPIError(err, p)
}
