package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
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

	state := regional.ResourceStateDeleting

	domain.Status = &regional.BlockStorageStatus{
		State: &state,
	}

	_, err := c.BlockStorageRepo.Update(ctx, domain)

	return err
}
