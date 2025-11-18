package network

import (
	"context"
	"log/slog"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"
	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/network"
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
	SKURepo port.ResourceQueryRepository[*regional.NetworkSKUDomain]
}

func (c ListSKUs) Do(ctx context.Context, tenantID string, params sdknetwork.ListSkusParams) (
	*sdknetwork.SkuIterator, error,
) {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	listParams := model.ListParams{
		Namespace: tenantID,
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}

	var domainSKUs []*regional.NetworkSKUDomain
	nextSkipToken, err := c.SKURepo.List(ctx, listParams, &domainSKUs)
	if err != nil {
		return nil, err
	}

	sdkNetworkSKUs := make([]sdkschema.NetworkSku, len(domainSKUs))
	for i := range domainSKUs {
		sdkNetworkSKUs[i] = *network.SkuToAPI(domainSKUs[i])
	}

	iterator := sdknetwork.SkuIterator{
		Items: sdkNetworkSKUs,
		Metadata: sdkschema.ResponseMetadata{
			Provider: ProviderNetworkName,
			Resource: skuv1.NetworkSKUResource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return &iterator, nil
}
