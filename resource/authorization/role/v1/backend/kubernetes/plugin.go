package kubernetes

import (
	"context"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
)

// RolePlugin is implemented by CSP plugins that manage role resources.
type RolePlugin interface {
	Create(ctx context.Context, resource *roledom.Role) error
	Delete(ctx context.Context, resource *roledom.Role) error
}
