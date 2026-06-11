package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type DeleteBlockStorage struct {
	Logger           *slog.Logger
	BlockStorageRepo port.WriterRepo[*regional.BlockStorageDomain]
}

func (c DeleteBlockStorage) Do(ctx context.Context, ir port.IdentifiableResource) error {
	domain := &regional.BlockStorageDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.Workspace = ir.GetWorkspace()
	domain.ResourceVersion = ir.GetVersion()

	return c.BlockStorageRepo.Delete(ctx, domain)
}
