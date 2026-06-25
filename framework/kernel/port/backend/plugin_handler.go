package backend

import (
	"context"
	"errors"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// ErrStillProcessing is returned when an operation is still in progress
// and the caller should requeue.
var ErrStillProcessing = errors.New("operation still in progress")

// PluginHandler defines the contract for handling resource-specific logic.
//
// It is intended to be implemented for each resource type that the delegator manages.
type PluginHandler[T persistence.IdentifiableResource] interface {
	// HandleAdmission validates a resource during admission control. It is
	// designed to be a hook that can reject a resource creation or update
	// based on defined policies or conditions.
	HandleAdmission(ctx context.Context, resource T) error

	// HandleReconcile processes the desired state of a resource and drives it
	// towards the current state. This is the core of the reconciliation loop
	// for a resource.
	//
	// TODO: Is the boolean return for requeue sufficient? Or do we want to return a duration for requeue after?
	// ISSUE: https://github.com/eu-sovereign-cloud/ecp/issues/187
	HandleReconcile(ctx context.Context, resource T) (requeue bool, err error)
}

// DelegatedFunc is a function type representing an operation delegated to a
// CSP plugin. It receives a context and the resource to operate on.
type DelegatedFunc[T persistence.IdentifiableResource] func(ctx context.Context, resource T) error

// RejectionConditionFunc is a function type that defines a condition for
// rejecting a resource. It should return an error if an unwanted condition is
// detected (e.g., decreasing the size of a block storage).
type RejectionConditionFunc[T persistence.IdentifiableResource] func(ctx context.Context, resource T) error
