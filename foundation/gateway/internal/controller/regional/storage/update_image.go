package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type UpdateImage struct {
	Logger    *slog.Logger
	ImageRepo port.WriterRepo[*regional.ImageDomain]
}

func (c UpdateImage) Do(
	ctx context.Context, domain *regional.ImageDomain,
) (*regional.ImageDomain, error) {
	result, err := c.ImageRepo.Update(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
