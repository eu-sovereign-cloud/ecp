package kubernetes

import (
	"fmt"
	"testing"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

func TestK8sToDomainErrorKind(t *testing.T) {
	gr := schema.GroupResource{Group: "test.io", Resource: "widgets"}

	tests := []struct {
		name     string
		err      error
		wantKind model.ErrKind
	}{
		{
			name:     "resource version conflict maps to PreconditionFailed",
			err:      kerrs.NewConflict(gr, "my-widget", fmt.Errorf("the object has been modified")),
			wantKind: model.KindPreconditionFailed,
		},
		{
			name: "field manager conflict maps to Conflict",
			err: kerrs.NewApplyConflict([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldManagerConflict,
					Message: "conflict with manager",
					Field:   ".spec.replicas",
				},
			}, "Apply failed with 1 conflict"),
			wantKind: model.KindConflict,
		},
		{
			name:     "not found",
			err:      kerrs.NewNotFound(gr, "my-widget"),
			wantKind: model.KindNotFound,
		},
		{
			name:     "already exists",
			err:      kerrs.NewAlreadyExists(gr, "my-widget"),
			wantKind: model.KindAlreadyExists,
		},
		{
			name:     "invalid",
			err:      kerrs.NewInvalid(schema.GroupKind{Group: "test.io", Kind: "Widget"}, "my-widget", nil),
			wantKind: model.KindValidation,
		},
		{
			name:     "forbidden",
			err:      kerrs.NewForbidden(gr, "my-widget", fmt.Errorf("access denied")),
			wantKind: model.KindForbidden,
		},
		{
			name:     "unauthorized",
			err:      kerrs.NewUnauthorized("not authenticated"),
			wantKind: model.KindForbidden,
		},
		{
			name:     "service unavailable",
			err:      kerrs.NewServiceUnavailable("backend down"),
			wantKind: model.KindUnavailable,
		},
		{
			name:     "unknown error falls back to unavailable",
			err:      fmt.Errorf("something unexpected"),
			wantKind: model.KindUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := k8sToDomainErrorKind(tt.err)
			if got != tt.wantKind {
				t.Errorf("k8sToDomainErrorKind() = %v, want %v", got, tt.wantKind)
			}
		})
	}
}

func TestKubeToDomainError(t *testing.T) {
	gr := schema.GroupResource{Group: "test.io", Resource: "widgets"}

	t.Run("nil error returns nil", func(t *testing.T) {
		if got := kubeToDomainError(nil); got != nil {
			t.Errorf("kubeToDomainError(nil) = %v, want nil", got)
		}
	})

	t.Run("resource version conflict produces PreconditionFailed error", func(t *testing.T) {
		err := kerrs.NewConflict(gr, "w", fmt.Errorf("modified"))
		domainErr := kubeToDomainError(err)
		if domainErr == nil {
			t.Fatal("expected non-nil error")
		}
		if domainErr.Kind != model.KindPreconditionFailed {
			t.Errorf("Kind = %v, want %v", domainErr.Kind, model.KindPreconditionFailed)
		}
	})

	t.Run("apply conflict produces Conflict error", func(t *testing.T) {
		err := kerrs.NewApplyConflict([]metav1.StatusCause{
			{Type: metav1.CauseTypeFieldManagerConflict, Message: "conflict"},
		}, "apply failed")
		domainErr := kubeToDomainError(err)
		if domainErr == nil {
			t.Fatal("expected non-nil error")
		}
		if domainErr.Kind != model.KindConflict {
			t.Errorf("Kind = %v, want %v", domainErr.Kind, model.KindConflict)
		}
	})
}
