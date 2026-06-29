package backend

import (
	"time"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// ConditionFromState creates a domain.StatusCondition from a resource state,
// supplying a standard message for each well-known lifecycle state.
func ConditionFromState(state domain.ResourceState) domain.StatusCondition {
	var message string
	switch state { //nolint:exhaustive // domain.ResourceStateError is handled by ConditionFromError.
	case domain.ResourceStatePending:
		message = "Resource is pending initialization."
	case domain.ResourceStateCreating:
		message = "Resource is being created."
	case domain.ResourceStateActive:
		message = "Resource is active and ready."
	case domain.ResourceStateUpdating:
		message = "Resource is being updated."
	case domain.ResourceStateDeleting:
		message = "Resource is being deleted."
	}
	return domain.StatusCondition{
		LastTransitionAt: time.Now(),
		Type:             "Reconcile",
		State:            state,
		Reason:           string(state),
		Message:          message,
	}
}

// ConditionFromError creates a domain.StatusCondition representing a reconciliation error.
func ConditionFromError(err error) domain.StatusCondition {
	return domain.StatusCondition{
		LastTransitionAt: time.Now(),
		Type:             "ReconcileError",
		State:            domain.ResourceStateError,
		Reason:           "ReconcileError",
		Message:          err.Error(),
	}
}

// DependencyPendingCondition creates a domain.StatusCondition reporting that a resource
// is waiting for a referenced dependency to become active before it can be created. The
// resource keeps state so its reconciliation re-evaluates the dependency on each requeue.
func DependencyPendingCondition(state domain.ResourceState, message string) domain.StatusCondition {
	return domain.StatusCondition{
		LastTransitionAt: time.Now(),
		Type:             "DependencyPending",
		State:            state,
		Reason:           "WaitingForDependency",
		Message:          message,
	}
}

// DeletionBlockedCondition creates a domain.StatusCondition reporting that a resource
// cannot be deleted while other resources still reference it. The resource keeps state
// (and therefore its cleanup finalizer) until the referrers are gone.
func DeletionBlockedCondition(state domain.ResourceState, message string) domain.StatusCondition {
	return domain.StatusCondition{
		LastTransitionAt: time.Now(),
		Type:             "DeletionBlocked",
		State:            state,
		Reason:           "ReferencedByDependent",
		Message:          message,
	}
}
