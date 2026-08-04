package kubernetes

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

// BlockStoragePlugin is implemented by CSP plugins that manage block storage resources.
type BlockStoragePlugin interface {
	Create(ctx context.Context, resource *bsdom.BlockStorage) error
	Delete(ctx context.Context, resource *bsdom.BlockStorage) error

	// Update reconciles an already-created resource towards its current spec. It is
	// level-triggered: called on every reconcile of an active resource, so it must be idempotent
	// and must not write when nothing has drifted. Full contract in doc/PLUGINS.md.
	Update(ctx context.Context, resource *bsdom.BlockStorage) error
	IncreaseSize(ctx context.Context, resource *bsdom.BlockStorage) error
}
