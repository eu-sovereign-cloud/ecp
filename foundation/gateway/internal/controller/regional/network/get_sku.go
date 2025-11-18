package network

import (
	"context"
	"log/slog"

	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/network"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type GetSKU struct {
	Logger  *slog.Logger
	SKURepo port.ResourceQueryRepository[*regional.NetworkSKUDomain]
}

func (c GetSKU) Do(
	ctx context.Context, tenantID, skuID string,
) (*sdkschema.NetworkSku, error) {
	domain := &regional.NetworkSKUDomain{}
	domain.SetName(skuID)
	// scope by tenant namespace if needed
	domain.SetNamespace(tenantID)
	if err := c.SKURepo.Load(ctx, &domain); err != nil {
		return nil, err
	}
	return network.SkuToAPI(domain), nil
}
