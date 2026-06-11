package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type GetBlockStorage struct {
	Logger           *slog.Logger
	BlockStorageRepo port.ReaderRepo[*regional.BlockStorageDomain]
}

func (c GetBlockStorage) Do(
	ctx context.Context, ir port.IdentifiableResource,
) (*regional.BlockStorageDomain, error) {
	domain := &regional.BlockStorageDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.Workspace = ir.GetWorkspace()
	domain.ResourceVersion = ir.GetVersion()

	if err := c.BlockStorageRepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
