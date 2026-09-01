package backend

import (
	"context"
	"errors"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// Failures a CSP plugin returns to tell the handler how to treat a rejected operation. Anything
// else is assumed transient: the failure is recorded and the operation is retried on a later pass.
//
// Progress signals are a separate family and live in requeue.go: StillProcessing, Revisit and
// RevisitBecause are not failures, they say the pass advanced without settling the resource.
var (
	// ErrNotSupported is returned when the provider cannot perform the operation at all - not
	// "not yet", but never, for this resource in this state. Retrying would re-issue a request
	// the provider has already rejected, so the handler records the reason and stops instead of
	// spinning. Most providers hit this on update: cloud resources routinely have immutable
	// fields (an Aruba VPC's region, an instance's flavor) that no amount of retrying will move.
	//
	// Wrap it to explain what was refused, so the reason reaches the resource's status:
	//
	//	fmt.Errorf("%w: an Aruba VPC's region cannot be changed after creation", backend.ErrNotSupported)
	ErrNotSupported = errors.New("operation not supported by the provider")
)

// PluginHandler defines the contract for handling resource-specific logic.
//
// It is intended to be implemented for each resource type that the delegator manages.
type PluginHandler[T persistence.IdentifiableResource] interface {
	// HandleAdmission validates a resource during admission control. It is
	// designed to be a hook that can reject a resource creation or update
	// based on defined policies or conditions.
	HandleAdmission(ctx context.Context, resource T) error

	// HandleReconcile drives the resource toward its desired state. This is the core of the
	// reconciliation loop for a resource.
	//
	// A nil error means the resource has reached a settled state and no requeue is needed.
	//
	// A RequeueError (StillProcessing, Revisit, RevisitBecause) is not a failure: the pass made
	// progress and the controller reschedules after the requested duration, or after the
	// configured default when it is zero.
	//
	// Any other non-nil error is a failure: it is recorded on the resource status and retried
	// with exponential backoff. An error wrapping ErrNotSupported stops the retry instead.
	//
	// A progress signal must be the outermost error. Never wrap a failure inside one - the
	// controller classifies failures first precisely because a wrapped one stays discoverable.
	HandleReconcile(ctx context.Context, resource T) error
}

// DelegatedFunc is a function type representing an operation delegated to a
// CSP plugin. It receives a context and the resource to operate on.
type DelegatedFunc[T persistence.IdentifiableResource] func(ctx context.Context, resource T) error

// RejectionConditionFunc is a function type that defines a condition for
// rejecting a resource. It should return an error if an unwanted condition is
// detected (e.g., decreasing the size of a block storage).
type RejectionConditionFunc[T persistence.IdentifiableResource] func(ctx context.Context, resource T) error
