package storage

import (
	"context"
	"log/slog"

	model "github.com/eu-sovereign-cloud/ecp/foundation/models"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"
)

type ListSKUs struct {
	Logger  *slog.Logger
	SKURepo port.ReaderRepo[*regional.StorageSKUDomain]
}

func (c ListSKUs) Do(ctx context.Context, params model.ListParams) (
	[]*regional.StorageSKUDomain, *string, error,
) {
	var domainSKUs []*regional.StorageSKUDomain
	nextSkipToken, err := c.SKURepo.List(ctx, params, &domainSKUs)
	if err != nil {
		return nil, nil, err
	}

	return domainSKUs, nextSkipToken, nil
}
