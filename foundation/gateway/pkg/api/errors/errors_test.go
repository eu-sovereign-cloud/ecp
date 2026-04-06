package errors

import (
	"net/http"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

func TestMapKindToHTTP(t *testing.T) {
	tests := []struct {
		name       string
		kind       model.ErrKind
		wantStatus int
		wantType   schema.ErrorType
	}{
		{
			name:       "Forbidden",
			kind:       model.KindForbidden,
			wantStatus: http.StatusForbidden,
			wantType:   schema.ErrorTypeForbidden,
		},
		{
			name:       "NotFound",
			kind:       model.KindNotFound,
			wantStatus: http.StatusNotFound,
			wantType:   schema.ErrorTypeResourceNotFound,
		},
		{
			name:       "Conflict",
			kind:       model.KindConflict,
			wantStatus: http.StatusConflict,
			wantType:   schema.ErrorTypeResourceConflict,
		},
		{
			name:       "AlreadyExists",
			kind:       model.KindAlreadyExists,
			wantStatus: http.StatusConflict,
			wantType:   schema.ErrorTypeResourceConflict,
		},
		{
			name:       "PreconditionFailed",
			kind:       model.KindPreconditionFailed,
			wantStatus: http.StatusPreconditionFailed,
			wantType:   schema.ErrorTypePreconditionFailed,
		},
		{
			name:       "Validation",
			kind:       model.KindValidation,
			wantStatus: http.StatusUnprocessableEntity,
			wantType:   schema.ErrorTypeValidationError,
		},
		{
			name:       "Unavailable",
			kind:       model.KindUnavailable,
			wantStatus: http.StatusInternalServerError,
			wantType:   schema.ErrorTypeInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, _, gotType := mapKindToHTTP(tt.kind)
			if gotStatus != tt.wantStatus {
				t.Errorf("status = %d, want %d", gotStatus, tt.wantStatus)
			}
			if gotType != tt.wantType {
				t.Errorf("type = %q, want %q", gotType, tt.wantType)
			}
		})
	}
}
