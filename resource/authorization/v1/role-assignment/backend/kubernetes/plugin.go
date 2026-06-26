package kubernetes

import (
	"context"

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// RoleAssignmentPlugin is implemented by CSP plugins that manage role assignment resources.
type RoleAssignmentPlugin interface {
	Create(ctx context.Context, resource *radom.RoleAssignment) error
	Delete(ctx context.Context, resource *radom.RoleAssignment) error
}
