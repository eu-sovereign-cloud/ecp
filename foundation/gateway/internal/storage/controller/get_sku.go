package controller

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/storage/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/storage/port"
)

type GetSKU struct {
	Repo port.SKURepo
}

func (c *GetSKU) Do(ctx context.Context, name string) (*model.SKU, error) {
	// TODO:
	// check access rights (policy)
	return c.Repo.GetByMetadataName(ctx, name)
}
