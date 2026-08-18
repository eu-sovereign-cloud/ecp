package kubernetes

import (
	"errors"
	"log/slog"

	kerrs "k8s.io/apimachinery/pkg/api/errors"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

// RetryLevel returns the level an error deserves when the caller is a control loop that will
// retry it.
//
// A conflict there means "not yet", not "broken": a resourceVersion moved under an update, or a
// namespace still holds children that are themselves terminating. The reconcile is requeued with
// backoff and converges on its own, usually within seconds — so logging it at ERROR turns routine
// convergence into something that looks pageable, and buries the failures that are not. Those keep
// ERROR.
//
// Both spellings are matched because these sites straddle the conversion boundary: some log the
// raw Kubernetes error, others the domain error it has already been wrapped into.
func RetryLevel(err error) slog.Level {
	if domainErr := kernel.AsError(err); domainErr != nil {
		switch domainErr.Kind {
		case kernel.KindConflict, kernel.KindPreconditionFailed:
			return slog.LevelWarn
		default:
			// Every other kind is decided below, not here: a domain error can still wrap a raw
			// Kubernetes conflict, so this falls through to the check rather than concluding
			// ERROR early. Listing the remaining kinds individually would be a lie — they all
			// take the same path.
		}
	}
	if kerrs.IsConflict(err) {
		return slog.LevelWarn
	}
	return slog.LevelError
}

// kubeToDomainError converts a Kubernetes API error to a domain Error.
func kubeToDomainError(err error) *kernel.Error {
	if err == nil {
		return nil
	}

	kind := k8sToDomainErrorKind(err)
	return kernel.NewError(kind, err)
}

// k8sToDomainErrorKind maps the standard Kubernetes error values to domain error kinds.
func k8sToDomainErrorKind(err error) kernel.ErrKind {
	switch {
	case kerrs.IsNotFound(err):
		return kernel.KindNotFound
	case kerrs.IsAlreadyExists(err):
		return kernel.KindAlreadyExists
	case kerrs.IsConflict(err):
		if isResourceVersionConflict(err) {
			return kernel.KindPreconditionFailed
		}
		return kernel.KindConflict
	case kerrs.IsInvalid(err):
		return kernel.KindValidation
	case kerrs.IsForbidden(err):
		return kernel.KindForbidden
	case kerrs.IsResourceExpired(err), kerrs.IsGone(err):
		return kernel.KindPreconditionFailed
	case kerrs.IsUnauthorized(err):
		return kernel.KindForbidden
	case kerrs.IsServiceUnavailable(err), kerrs.IsServerTimeout(err), kerrs.IsTimeout(err):
		return kernel.KindUnavailable
	default:
		return kernel.KindUnavailable
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
