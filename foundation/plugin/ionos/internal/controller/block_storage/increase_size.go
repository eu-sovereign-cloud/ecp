package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/pkg/port"
)

type IncreaseSizeBlockStorage struct {
	Store port.BlockStorageStore
}

func (i *IncreaseSizeBlockStorage) Do(ctx context.Context, domain *regional.BlockStorageDomain) error {
	return i.Store.IncreaseSize(ctx, domain)
}
