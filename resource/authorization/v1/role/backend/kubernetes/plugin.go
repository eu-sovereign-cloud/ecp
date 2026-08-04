package kubernetes

import (
	"context"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
)

// RolePlugin is implemented by CSP plugins that manage role resources.
type RolePlugin interface {
	Create(ctx context.Context, resource *roledom.Role) error
	Delete(ctx context.Context, resource *roledom.Role) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *roledom.Role) error
}
