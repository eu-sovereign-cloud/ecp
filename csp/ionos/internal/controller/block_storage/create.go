package block_storage

import (
	"context"

	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/domain"
	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorageDomain) error {
	return c.Store.Create(ctx, domain)
}
