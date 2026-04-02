package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	delegato "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

// GenericPluginHandler provides a generic implementation of the
// DelegatorPluginHandler interface. It can be used as a base for specific
// resource handlers, allowing for composition of rejection conditions and
// operations.
type GenericPluginHandler[T gateway.IdentifiableResource] struct {
	rejectionConditions []delegato.RejectionConditionFunc[T]
}

// NewPluginHandler creates a new GenericPluginHandler with the
// provided rejection conditions and resource operations.
func NewPluginHandler[T gateway.IdentifiableResource](
	rejectionConditions []delegato.RejectionConditionFunc[T],
) *GenericPluginHandler[T] {
	return &GenericPluginHandler[T]{
		rejectionConditions: rejectionConditions,
	}
}

// SetRejectionConditions replaces the existing rejection conditions with a new
// set.
func (h *GenericPluginHandler[T]) SetRejectionConditions(rejectionConditions ...delegato.RejectionConditionFunc[T]) {
	h.rejectionConditions = rejectionConditions
}

// AddRejectionConditions appends additional rejection conditions to the
// existing list.
func (h *GenericPluginHandler[T]) AddRejectionConditions(rejectionConditions ...delegato.RejectionConditionFunc[T]) {
	h.rejectionConditions = append(h.rejectionConditions, rejectionConditions...)
}

// HandleAdmission iterates through the configured rejection conditions and
// returns a joined error if any of the conditions fail. This is used to
// validate a resource before it is persisted.
//
// Note: This is an generic implementation of the logic of the
// `HandleStorage.AddmissionHook(...) error` in the proposal, but replacing the
// `switch-case` chain by a loop throw a rejection condition slice.
func (h *GenericPluginHandler[T]) HandleAdmission(ctx context.Context, resource T) error {
	var errs []error

	for _, condition := range h.rejectionConditions {
		if err := condition(ctx, resource); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("admission failed: %w", err)
	}

	return nil
}

func BypassDelegated[T gateway.IdentifiableResource](_ context.Context, _ T) error {
	return nil
}

// conditionFromState creates a new regional.StatusConditionDomain with standard values.
func conditionFromState(state regional.ResourceStateDomain) regional.StatusConditionDomain {
	var message string
	switch state { //nolint:exhaustive // regional.ResourceStateError is treated by a specific function.
	case regional.ResourceStatePending:
		message = "Resource is pending initialization."
	case regional.ResourceStateCreating:
		message = "Resource is being created."
	case regional.ResourceStateActive:
		message = "Resource is active and ready."
	case regional.ResourceStateUpdating:
		message = "Resource is being updated."
	case regional.ResourceStateDeleting:
		message = "Resource is being deleted."
	case regional.ResourceStateSuspended:
		message = "Resource is suspended."
	}

	return regional.StatusConditionDomain{
		LastTransitionAt: time.Now(),
		Type:             string(state),
		State:            state,
		Reason:           string(state),
		Message:          message,
	}
}

// conditionFromError creates a new regional.StatusConditionDomain with an error state and message.
func conditionFromError(err error) regional.StatusConditionDomain {
	return regional.StatusConditionDomain{
		LastTransitionAt: time.Now(),
		Type:             string(regional.ResourceStateError),
		State:            regional.ResourceStateError,
		Reason:           "ReconcileError", // A generic reason for reconciliation failures
		Message:          err.Error(),
	}
}
