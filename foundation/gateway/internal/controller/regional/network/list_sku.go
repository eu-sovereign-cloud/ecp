package network

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

const (
	NetworkBaseURL      = "/providers/seca.network"
	ProviderNetworkName = "seca.network/v1"
)

type ListSKUs struct {
	Logger  *slog.Logger
	SKURepo port.ReaderRepo[*regional.NetworkSKUDomain]
}

func (c ListSKUs) Do(ctx context.Context, tenantID string, params model.ListParams) (
	[]*regional.NetworkSKUDomain, *string, error,
) {
	params.SetTenant(tenantID)

	var domainSKUs []*regional.NetworkSKUDomain
	nextSkipToken, err := c.SKURepo.List(ctx, params, &domainSKUs)
	if err != nil {
		return nil, nil, err
	}

	return domainSKUs, nextSkipToken, nil
}
