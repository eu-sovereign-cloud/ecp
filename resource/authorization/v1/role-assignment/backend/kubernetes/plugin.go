package kubernetes

import (
	"context"

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// RoleAssignmentPlugin is implemented by CSP plugins that manage role assignment resources.
type RoleAssignmentPlugin interface {
	Create(ctx context.Context, resource *radom.RoleAssignment) error
	Delete(ctx context.Context, resource *radom.RoleAssignment) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *radom.RoleAssignment) error
}
