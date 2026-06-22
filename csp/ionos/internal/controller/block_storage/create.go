package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorageDomain) error {
	return c.Store.Create(ctx, domain)
}
