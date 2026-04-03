package kubernetes

import (
	"errors"

	kerrs "k8s.io/apimachinery/pkg/api/errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// kubeToDomainError converts a Kubernetes API error to a domain Error.
func kubeToDomainError(err error) *model.Error {
	if err == nil {
		return nil
	}

	kind := k8sToDomainErrorKind(err)
	return model.NewError(kind, err)
}

// k8sToDomainErrorKind maps the standard Kubernetes error values to domain error kinds.
func k8sToDomainErrorKind(err error) model.ErrKind {
	switch {
	case kerrs.IsNotFound(err):
		return model.KindNotFound
	case kerrs.IsAlreadyExists(err):
		return model.KindAlreadyExists
	case kerrs.IsConflict(err):
		if isResourceVersionConflict(err) {
			return model.KindPreconditionFailed
		}
		return model.KindConflict
	case kerrs.IsInvalid(err):
		return model.KindValidation
	case kerrs.IsForbidden(err):
		return model.KindForbidden
	case kerrs.IsResourceExpired(err), kerrs.IsGone(err):
		return model.KindPreconditionFailed
	case kerrs.IsUnauthorized(err):
		return model.KindForbidden
	case kerrs.IsServiceUnavailable(err), kerrs.IsServerTimeout(err), kerrs.IsTimeout(err):
		return model.KindUnavailable
	default:
		return model.KindUnavailable
	}
}

// isResourceVersionConflict verifies if the returned 409 Conflict is due to a resourceVersion mismatch or is an
// actual apply conflict
func isResourceVersionConflict(err error) bool {
	if !kerrs.IsConflict(err) {
		return false
	}

	statusErr, ok := errors.AsType[*kerrs.StatusError](err)
	if !ok {
		return false
	}

	// resourceVersion conflicts do not have causes in the status details
	if statusErr.Status().Details == nil || len(statusErr.Status().Details.Causes) == 0 {
		return true
	}

	return false
}
