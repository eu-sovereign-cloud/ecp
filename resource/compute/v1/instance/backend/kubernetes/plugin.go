package kubernetes

import (
	"context"

	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

// InstancePlugin is implemented by CSP plugins that manage Instance resources.
type InstancePlugin interface {
	Create(ctx context.Context, resource *instancedom.Instance) error
	Delete(ctx context.Context, resource *instancedom.Instance) error

	// Update reconciles an already-created resource towards its current spec. It is called on
	// every reconcile of an active resource rather than on a detected change, so it must be
	// idempotent and cheap when nothing has drifted - compare against the provider and return
	// nil if there is nothing to do.
	//
	// Return backend.ErrStillProcessing while the change is in flight. Return an error wrapping
	// backend.ErrNotSupported when the provider cannot apply the change at all (an immutable
	// field, say); the reason is reported on the resource and the call is not retried. Any other
	// error is treated as transient and retried.
	Update(ctx context.Context, resource *instancedom.Instance) error
	// PowerOn transitions an instance to the powered-on state.
	PowerOn(ctx context.Context, resource *instancedom.Instance) error
	// PowerOff transitions an instance to the powered-off state.
	PowerOff(ctx context.Context, resource *instancedom.Instance) error
}
