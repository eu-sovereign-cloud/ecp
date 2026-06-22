package port

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

type BlockStorageStore interface {
	Create(ctx context.Context, domain *bsdom.BlockStorageDomain) error
	Delete(ctx context.Context, domain *bsdom.BlockStorageDomain) error
	IncreaseSize(ctx context.Context, domain *bsdom.BlockStorageDomain) error
}
