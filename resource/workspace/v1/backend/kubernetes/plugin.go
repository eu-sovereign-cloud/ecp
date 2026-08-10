package kubernetes

import (
	"context"

	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// WorkspacePlugin is implemented by CSP plugins that manage workspace resources.
type WorkspacePlugin interface {
	Create(ctx context.Context, resource *wsdom.Workspace) error
	Delete(ctx context.Context, resource *wsdom.Workspace) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *wsdom.Workspace) error
}
