package kubernetes

import (
	"context"

	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// ImagePlugin is implemented by CSP plugins that manage image resources.
type ImagePlugin interface {
	Create(ctx context.Context, resource *imgdom.Image) error
	Delete(ctx context.Context, resource *imgdom.Image) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *imgdom.Image) error
}
