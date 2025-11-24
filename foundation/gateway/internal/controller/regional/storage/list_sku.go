package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type ListSKUs struct {
	Logger  *slog.Logger
	SKURepo port.ResourceQueryRepository[*regional.StorageSKUDomain]
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
