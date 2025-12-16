package port

import (
	"context"
	"errors"
	"fmt"

	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// DelegatorResourceHandler defines the contract for handling resource-specific
// logic.
//
// It is intended to be implemented for each resource type that the delegator
// manages.
type DelegatorResourceHandler[T gateway_port.NamespacedResource] interface {
	// HandleAdmission validates a resource during admission control. It's
	// designed to be a hook that can reject a resource creation or update
	// based on defined policies or conditions.
	//
	// Note: It corresponds to the `HandleStorage.AddmissionHook(...) error` in
	// the ptoposal.
	HandleAdmission(ctx context.Context, resource T) error

	// HandleReconcile processes the desired state of a resource and drives it
	// towards the current state. This is the core of the reconciliation loop
	// for a resource.
	//
	// Note: It corresponds to the `HandleStorage.Do(...) error` in the
	// ptoposal.
	HandleReconcile(ctx context.Context, resource T) error
}

// RejectionConditionFunc is a function type that defines a condition for
// rejecting a resource. It should return an error if an unwanted condition is
// detected (e.g., decreasing the size of a block storage).
type RejectionConditionFunc[T gateway_port.NamespacedResource] func(ctx context.Context, resource T) error

// GenericDelegatorResourceHandler provides a generic implementation of the
// DelegatorResourceHandler interface. It can be used as a base for specific
// resource handlers, allowing for composition of rejection conditions and
// operations.
type GenericDelegatorResourceHandler[T gateway_port.NamespacedResource] struct {
	rejectionConditions []RejectionConditionFunc[T]
}

// NewResourceHandler creates a new GenericDelegatorResourceHandler with the
// provided rejection conditions and resource operations.
func NewResourceHandler[T gateway_port.NamespacedResource](
	rejectionConditions []RejectionConditionFunc[T],
) *GenericDelegatorResourceHandler[T] {
	return &GenericDelegatorResourceHandler[T]{
		rejectionConditions: rejectionConditions,
	}
}

// SetRejectionConditions replaces the existing rejection conditions with a new
// set.
func (h *GenericDelegatorResourceHandler[T]) SetRejectionConditions(rejectionConditions ...RejectionConditionFunc[T]) {
	h.rejectionConditions = rejectionConditions
}

// AddRejectionConditions appends additional rejection conditions to the
// existing list.
func (h *GenericDelegatorResourceHandler[T]) AddRejectionConditions(rejectionConditions ...RejectionConditionFunc[T]) {
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
