package kubernetes

import (
	"context"

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
)

// RoleAssignmentPlugin is implemented by CSP plugins that manage role assignment resources.
type RoleAssignmentPlugin interface {
	Create(ctx context.Context, resource *radom.RoleAssignment) error
	Delete(ctx context.Context, resource *radom.RoleAssignment) error
}
