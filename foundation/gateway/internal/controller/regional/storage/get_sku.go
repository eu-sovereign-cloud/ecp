package storage

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type GetSKU struct {
	Logger  *slog.Logger
	SKURepo port.ReaderRepo[*regional.StorageSKUDomain]
}

func (c GetSKU) Do(
	ctx context.Context, ir port.IdentifiableResource,
) (*regional.StorageSKUDomain, error) {
	domain := &regional.StorageSKUDomain{}
	domain.Name = ir.GetName()
	domain.Tenant = ir.GetTenant() // ensure namespaced SKU retrieval
	if err := c.SKURepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
