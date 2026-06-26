package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

type IncreaseSizeBlockStorage struct {
	Store port.BlockStorageStore
}

func (i *IncreaseSizeBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorage) error {
	return i.Store.IncreaseSize(ctx, domain)
}
