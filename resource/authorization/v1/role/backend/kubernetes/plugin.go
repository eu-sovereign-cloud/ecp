package kubernetes

import (
	"context"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
)

// RolePlugin is implemented by CSP plugins that manage role resources.
type RolePlugin interface {
	Create(ctx context.Context, resource *roledom.Role) error
	Delete(ctx context.Context, resource *roledom.Role) error

	// Update reconciles an already-created resource towards its current spec. It is called on
	// every reconcile of an active resource rather than on a detected change, so it must be
	// idempotent and cheap when nothing has drifted - compare against the provider and return
	// nil if there is nothing to do.
	//
	// Return backend.ErrStillProcessing while the change is in flight. Return an error wrapping
	// backend.ErrNotSupported when the provider cannot apply the change at all (an immutable
	// field, say); the reason is reported on the resource and the call is not retried. Any other
	// error is treated as transient and retried.
	Update(ctx context.Context, resource *roledom.Role) error
}
