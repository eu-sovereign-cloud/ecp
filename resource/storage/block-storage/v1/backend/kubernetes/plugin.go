package kubernetes

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
)

// BlockStoragePlugin is implemented by CSP plugins that manage block storage resources.
type BlockStoragePlugin interface {
	Create(ctx context.Context, resource *bsdom.BlockStorage) error
	Delete(ctx context.Context, resource *bsdom.BlockStorage) error
	IncreaseSize(ctx context.Context, resource *bsdom.BlockStorage) error
}
