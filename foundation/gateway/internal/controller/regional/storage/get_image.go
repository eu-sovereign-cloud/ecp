package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type GetImage struct {
	Logger    *slog.Logger
	ImageRepo port.ReaderRepo[*regional.ImageDomain]
}

func (c GetImage) Do(
	ctx context.Context, ir port.IdentifiableResource,
) (*regional.ImageDomain, error) {
	domain := &regional.ImageDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant()
	domain.ResourceVersion = ir.GetVersion()

	if err := c.ImageRepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
