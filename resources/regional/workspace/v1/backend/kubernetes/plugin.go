package kubernetes

import (
	"context"

	wsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/workspace/v1"
)

// WorkspacePlugin is implemented by CSP plugins that manage workspace resources.
type WorkspacePlugin interface {
	Create(ctx context.Context, resource *wsdom.WorkspaceDomain) error
	Delete(ctx context.Context, resource *wsdom.WorkspaceDomain) error
}
