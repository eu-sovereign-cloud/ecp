package handler

import (
	"errors"
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// mapCreateErrorToHTTP maps domain-specific errors to HTTP status codes and messages for create operations
func mapCreateErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, model.ErrAlreadyExists):
		return http.StatusConflict, "resource already exists"
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, "parent resource not found"
	case errors.Is(err, model.ErrValidation):
		return http.StatusUnprocessableEntity, "resource validation failed"
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
	}
}

// mapUpdateErrorToHTTP maps domain-specific errors to HTTP status codes and messages for update operations
func mapUpdateErrorToHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, "resource not found"
	case errors.Is(err, model.ErrValidation):
		return http.StatusUnprocessableEntity, "resource validation failed"
	case errors.Is(err, model.ErrConflict):
		return http.StatusConflict, "resource update conflict"
	default:
		return http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)
	}
}
