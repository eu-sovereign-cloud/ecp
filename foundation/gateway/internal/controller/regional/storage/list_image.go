package storage

import (
	"context"
	"log/slog"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type ListImages struct {
	Logger    *slog.Logger
	ImageRepo port.ReaderRepo[*regional.ImageDomain]
}

func (c ListImages) Do(ctx context.Context, params model.ListParams) (
	[]*regional.ImageDomain, *string, error,
) {
	var domainImages []*regional.ImageDomain
	nextSkipToken, err := c.ImageRepo.List(ctx, params, &domainImages)
	if err != nil {
		return nil, nil, err
	}

	return domainImages, nextSkipToken, nil
}
