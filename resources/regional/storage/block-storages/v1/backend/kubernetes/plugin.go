package kubernetes

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

// BlockStoragePlugin is implemented by CSP plugins that manage block storage resources.
type BlockStoragePlugin interface {
	Create(ctx context.Context, resource *bsdom.BlockStorageDomain) error
	Delete(ctx context.Context, resource *bsdom.BlockStorageDomain) error
	IncreaseSize(ctx context.Context, resource *bsdom.BlockStorageDomain) error
}
