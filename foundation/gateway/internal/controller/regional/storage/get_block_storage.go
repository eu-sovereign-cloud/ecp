package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
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
	if err := c.BlockStorageRepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
