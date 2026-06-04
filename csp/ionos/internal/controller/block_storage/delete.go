package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/domain/regional"
)

type DeleteBlockStorage struct {
	Store port.BlockStorageStore
}

func (d *DeleteBlockStorage) Do(ctx context.Context, domain *regional.BlockStorageDomain) error {
	return d.Store.Delete(ctx, domain)
}
