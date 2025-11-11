package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/storage/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/storage/port"
)

type CreateSKU struct {
	Repo port.SKURepo
}

func (c *CreateSKU) Do(ctx context.Context, sku *model.SKU) error {
	// TODO:
	// check access rights (policy)
	// validation of sku syntax & semantics
	// maybe quota checks
	// activity logging of customer
	// rate limits
	// metrics, tracing, ...
	return c.Repo.Create(ctx, sku)
}
