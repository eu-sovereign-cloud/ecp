package backend

import (
	"time"

	"github.com/eu-sovereign-cloud/ecp/resources/common/domain"
)

// ConditionFromState creates a domain.StatusConditionDomain from a resource state,
// supplying a standard message for each well-known lifecycle state.
func ConditionFromState(state domain.ResourceStateDomain) domain.StatusConditionDomain {
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
	return domain.StatusConditionDomain{
		LastTransitionAt: time.Now(),
		Type:             "Reconcile",
		State:            state,
		Reason:           string(state),
		Message:          message,
	}
}

// ConditionFromError creates a domain.StatusConditionDomain representing a reconciliation error.
func ConditionFromError(err error) domain.StatusConditionDomain {
	return domain.StatusConditionDomain{
		LastTransitionAt: time.Now(),
		Type:             "ReconcileError",
		State:            domain.ResourceStateError,
		Reason:           "ReconcileError",
		Message:          err.Error(),
	}
}
