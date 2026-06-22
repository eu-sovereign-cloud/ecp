package backend

import (
	"context"
	"errors"
	"fmt"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// GenericPluginHandler provides a generic implementation of the PluginHandler admission check.
// Embed this into per-resource plugin handlers to inherit admission logic.
type GenericPluginHandler[T persistence.IdentifiableResource] struct {
	rejectionConditions []backendport.RejectionConditionFunc[T]
	MaxConditions       int
}

// NewGenericPluginHandler creates a new GenericPluginHandler with the provided rejection conditions.
func NewGenericPluginHandler[T persistence.IdentifiableResource](
	rejectionConditions []backendport.RejectionConditionFunc[T],
) *GenericPluginHandler[T] {
	return &GenericPluginHandler[T]{rejectionConditions: rejectionConditions}
}

// SetRejectionConditions replaces the existing rejection conditions with a new set.
func (h *GenericPluginHandler[T]) SetRejectionConditions(rcs ...backendport.RejectionConditionFunc[T]) {
	h.rejectionConditions = rcs
}

// AddRejectionConditions appends additional rejection conditions to the existing list.
func (h *GenericPluginHandler[T]) AddRejectionConditions(rcs ...backendport.RejectionConditionFunc[T]) {
	h.rejectionConditions = append(h.rejectionConditions, rcs...)
}

// HandleAdmission validates a resource during admission control by evaluating all configured
// rejection conditions and returning a joined error if any condition fails.
func (h *GenericPluginHandler[T]) HandleAdmission(ctx context.Context, resource T) error {
	var errs []error
	for _, cond := range h.rejectionConditions {
		if err := cond(ctx, resource); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("admission failed: %w", err)
	}
	return nil
}

// BypassDelegated is a no-op DelegatedFunc for state transitions that need no external call.
func BypassDelegated[T persistence.IdentifiableResource](_ context.Context, _ T) error {
	return nil
}
