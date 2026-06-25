package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorage) error {
	return c.Store.Create(ctx, domain)
}
