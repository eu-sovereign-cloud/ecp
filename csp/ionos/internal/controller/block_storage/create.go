package block_storage

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/port"
<<<<<<< HEAD
	github.com/eu-sovereign-cloud/ecp/foundation/models/domain/domain/regional"
=======
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
>>>>>>> 0b257c98 (refactor: moved kubernetes-related to foundation/persistence and rest-related to foundation/gateway)
)

type CreateBlockStorage struct {
	Store port.BlockStorageStore
}

func (c *CreateBlockStorage) Do(ctx context.Context, domain *regional.BlockStorageDomain) error {
	return c.Store.Create(ctx, domain)
}
