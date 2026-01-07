package port

import (
	"context"

	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// ResourceHandler defines the contract for handling resource-specific logic.
//
// It is intended to be implemented for each resource type that the delegator
// manages.
type ResourceHandler[T gateway_port.IdentifiableResource] interface {
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
type RejectionConditionFunc[T gateway_port.IdentifiableResource] func(ctx context.Context, resource T) error
