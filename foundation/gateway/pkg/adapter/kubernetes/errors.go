package kubernetes

import (
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
)

// adapterToDomainError converts a Kubernetes API error to a domain Error.
// It maps Kubernetes error types to the appropriate domain error kinds.
func adapterToDomainError(err error) *model.Error {
	if err == nil {
		return nil
	}

	kind := k8sToDomainErrorKind(err)
	return model.NewError(kind, err)
}

// k8sToDomainErrorKind determines the domain error kind from a Kubernetes error.
func k8sToDomainErrorKind(err error) model.ErrKind {
	switch {
	case kerrs.IsNotFound(err):
		return model.KindNotFound
	case kerrs.IsAlreadyExists(err):
		return model.KindAlreadyExists
	case kerrs.IsConflict(err):
		return model.KindConflict
	case kerrs.IsInvalid(err):
		return model.KindValidation
	case kerrs.IsForbidden(err):
		return model.KindForbidden
	case kerrs.IsUnauthorized(err):
		return model.KindForbidden
	case kerrs.IsServiceUnavailable(err), kerrs.IsServerTimeout(err), kerrs.IsTimeout(err):
		return model.KindUnavailable
	default:
		return model.KindUnavailable
	}
}
