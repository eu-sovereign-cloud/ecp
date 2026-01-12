package handler

import (
	"context"
	"errors"
	"fmt"

	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	delegator_port "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port"
)

// GenericDelegatorResourceHandler provides a generic implementation of the
// DelegatorResourceHandler interface. It can be used as a base for specific
// resource handlers, allowing for composition of rejection conditions and
// operations.
type GenericDelegatorResourceHandler[T gateway_port.IdentifiableResource] struct {
	rejectionConditions []delegator_port.RejectionConditionFunc[T]
}

// NewResourceHandler creates a new GenericDelegatorResourceHandler with the
// provided rejection conditions and resource operations.
func NewResourceHandler[T gateway_port.IdentifiableResource](
	rejectionConditions []delegator_port.RejectionConditionFunc[T],
) *GenericDelegatorResourceHandler[T] {
	return &GenericDelegatorResourceHandler[T]{
		rejectionConditions: rejectionConditions,
	}
}

// SetRejectionConditions replaces the existing rejection conditions with a new
// set.
func (h *GenericDelegatorResourceHandler[T]) SetRejectionConditions(rejectionConditions ...delegator_port.RejectionConditionFunc[T]) {
	h.rejectionConditions = rejectionConditions
}

// AddRejectionConditions appends additional rejection conditions to the
// existing list.
func (h *GenericDelegatorResourceHandler[T]) AddRejectionConditions(rejectionConditions ...delegator_port.RejectionConditionFunc[T]) {
	h.rejectionConditions = append(h.rejectionConditions, rejectionConditions...)
}

// HandleAdmission iterates through the configured rejection conditions and
// returns a joined error if any of the conditions fail. This is used to
// validate a resource before it is persisted.
//
// Note: This is an generic implementation of the logic of the
// `HandleStorage.AddmissionHook(...) error` in the proposal, but replacing the
// `switch-case` chain by a loop throw a rejection condition slice.
func (h *GenericDelegatorResourceHandler[T]) HandleAdmission(ctx context.Context, resource T) error {
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

func BypassDelegated[T gateway_port.IdentifiableResource](_ context.Context, _ T) error {
	return nil
}
