package rest

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

func TestMapKindToHTTP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		kind       kernel.ErrKind
		wantStatus int
	}{
		{"unauthorized", kernel.KindUnauthorized, http.StatusUnauthorized},
		{"forbidden", kernel.KindForbidden, http.StatusForbidden},
		{"not found", kernel.KindNotFound, http.StatusNotFound},
		{"conflict", kernel.KindConflict, http.StatusConflict},
		{"already exists", kernel.KindAlreadyExists, http.StatusConflict},
		{"precondition failed", kernel.KindPreconditionFailed, http.StatusPreconditionFailed},
		{"validation", kernel.KindValidation, http.StatusUnprocessableEntity},
		{"unavailable", kernel.KindUnavailable, http.StatusInternalServerError},
		{"internal", kernel.KindInternal, http.StatusInternalServerError},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _, _ := mapKindToHTTP(tc.kind)
			if got != tc.wantStatus {
				t.Errorf("mapKindToHTTP(%v) = %d, want %d", tc.kind, got, tc.wantStatus)
			}
		})
	}
}

func TestWriteErrorResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/some/path", nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	WriteErrorResponse(w, r, log, kernel.ErrUnauthorized)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
}

func TestWriteErrorResponse_BadRequest(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	WriteErrorResponse(w, r, log, errors.Join(ErrBadRequest, errors.New("field missing")))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
