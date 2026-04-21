package region

import (
	"fmt"
	"strconv"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/regions/v1"
	regionv1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/validation"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

func ListParamsFromAPI(params regionv1.ListRegionsParams) model.ListParams {
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
		sdkRegions[i] = domainToAPI(*dom, "list")
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

// DomainToAPI converts a RegionDomain to an API Region for Get operations
func DomainToAPI(domain *model.RegionDomain) schema.Region {
	return domainToAPI(*domain, "get")
}

// domainToAPI converts a RegionDomain to an API schema.Region, embedding metadata verb.
func domainToAPI(dom model.RegionDomain, verb string) schema.Region {
	providers := make([]schema.Provider, 0, len(dom.Providers))
	for _, p := range dom.Providers {
		providers = append(providers, schema.Provider{Name: p.Name, Url: p.URL, Version: p.Version})
	}
	zones := make([]schema.Zone, 0, len(dom.Zones))
	for _, z := range dom.Zones {
		zones = append(zones, schema.Zone(z))
	}
	resVersion := int64(0)
	// resourceVersion is best-effort numeric
	if rv, err := strconv.ParseInt(dom.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}
	sdk := schema.Region{
		Spec: schema.RegionSpec{
			AvailableZones: zones,
			Providers:      providers,
		},
		Metadata: &schema.GlobalResourceMetadata{
			ApiVersion:      regionsv1.Version,
			CreatedAt:       dom.CreatedAt,
			LastModifiedAt:  dom.UpdatedAt,
			Kind:            schema.GlobalResourceMetadataKindResourceKindRegion,
			Name:            dom.Name,
			Provider:        model.ProviderRegionName,
			Resource:        dom.Name,
			Ref:             fmt.Sprintf("%s/%s", model.RegionBaseURL, dom.Name), // ignore mapping error, not critical internally
			ResourceVersion: resVersion,
			Verb:            verb,
		},
	}
	if dom.DeletedAt != nil {
		sdk.Metadata.DeletedAt = dom.DeletedAt
	}
	return sdk
}
