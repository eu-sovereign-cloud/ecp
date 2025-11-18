package region

import (
	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

func ListParamsFromSDK(params regionv1.ListRegionsParams) model.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return model.ListParams{
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

func DomainToAPIIterator(domainRegions []*model.RegionDomain, nextSkipToken *string) *regionv1.RegionIterator {
	sdkRegions := make([]schema.Region, len(domainRegions))
	for i, dom := range domainRegions {
		sdkRegions[i] = model.MapRegionDomainToSDK(*dom, "list")
	}

	iterator := &regionv1.RegionIterator{
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

	return iterator
}
