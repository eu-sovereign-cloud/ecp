package errors

import (
	"errors"
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// ModelToHTTPError maps domain-specific errors to HTTP status codes and messages
func ModelToHTTPError(err error) (int, string) {
	switch {
	case errors.Is(err, model.ErrAlreadyExists):
		return http.StatusConflict, "resource already exists"
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, model.ErrValidation):
		return http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, model.ErrConflict):
		return http.StatusPreconditionFailed, "resource precondition failed"
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
	}
}
