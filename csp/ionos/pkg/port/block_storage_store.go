package port

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/storage/block-storages/v1"
)

type BlockStorageStore interface {
	Create(ctx context.Context, domain *bsdom.BlockStorage) error
	Delete(ctx context.Context, domain *bsdom.BlockStorage) error
	IncreaseSize(ctx context.Context, domain *bsdom.BlockStorage) error
}
