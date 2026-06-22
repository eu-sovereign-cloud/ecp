package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1"
)

type DeleteBlockStorage struct {
	Store port.BlockStorageStore
}

func (d *DeleteBlockStorage) Do(ctx context.Context, domain *bsdom.BlockStorage) error {
	return d.Store.Delete(ctx, domain)
}
