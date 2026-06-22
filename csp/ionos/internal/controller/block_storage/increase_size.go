package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

type IncreaseSizeBlockStorage struct {
	Store port.BlockStorageStore
}

func (i *IncreaseSizeBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorage) error {
	return i.Store.IncreaseSize(ctx, domain)
}
