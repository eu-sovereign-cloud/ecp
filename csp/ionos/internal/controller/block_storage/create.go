package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	github.com/eu-sovereign-cloud/ecp/foundation/models/domain/domain/regional"
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *regional.BlockStorageDomain) error {
	return c.Store.Create(ctx, domain)
}
