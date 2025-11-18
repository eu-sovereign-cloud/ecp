package region

import (
	"context"
	"log/slog"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type ListRegion struct {
	Logger *slog.Logger
	Repo   port.ResourceQueryRepository[*model.RegionDomain]
}

// ListRegions retrieves all available regions, maps them to the domain, and then projects them to the SDK model.
func (c *ListRegion) Do(ctx context.Context, params region.ListRegionsParams) (*region.RegionIterator, error) {
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
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}

	var domainRegions []*model.RegionDomain
	nextSkipToken, err := c.Repo.List(ctx, listParams, &domainRegions)
	if err != nil {
		return nil, err
	}

	sdkRegions := make([]schema.Region, len(domainRegions))
	for i, dom := range domainRegions {
		sdkRegions[i] = model.MapRegionDomainToSDK(*dom, "list")
	}

	iterator := &region.RegionIterator{
		Items: sdkRegions,
		Metadata: schema.ResponseMetadata{
			Provider: model.ProviderRegionName,
			Resource: regionsv1.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator, nil
}
