package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type DeleteBlockStorage struct {
	Store port.BlockStorageStore
}

func (d *DeleteBlockStorage) Do(ctx context.Context, domain *regional.BlockStorageDomain) error {
	return d.Store.Delete(ctx, domain)
}
