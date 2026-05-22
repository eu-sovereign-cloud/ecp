package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *regional.BlockStorageDomain) error {
	return c.Store.Create(ctx, domain)
}
