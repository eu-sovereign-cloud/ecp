package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorage) error {
	return c.Store.Create(ctx, domain)
}
