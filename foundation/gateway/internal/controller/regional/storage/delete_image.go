package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type DeleteImage struct {
	Logger    *slog.Logger
	ImageRepo port.WriterRepo[*regional.ImageDomain]
}

func (c DeleteImage) Do(ctx context.Context, ir port.IdentifiableResource) error {
	domain := &regional.ImageDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.ResourceVersion = ir.GetVersion()

	return c.ImageRepo.Delete(ctx, domain)
}
