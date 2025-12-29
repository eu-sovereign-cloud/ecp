package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type GetSKU struct {
	Logger  *slog.Logger
	SKURepo port.ReaderRepo[*regional.NetworkSKUDomain]
}

func (c GetSKU) Do(
	ctx context.Context, tenantID, skuID string,
) (*regional.NetworkSKUDomain, error) {
	domain := &regional.NetworkSKUDomain{}
	domain.Name = skuID
	// scope by tenant namespace if needed
	domain.Tenant = tenantID
	if err := c.SKURepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return domain, nil
}
