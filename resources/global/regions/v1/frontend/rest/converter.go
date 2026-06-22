// Package rest provides REST↔domain conversion and HTTP handlers for the region resource.
package rest

import (
	"fmt"
	"strconv"

	regionv1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	rdom "github.com/eu-sovereign-cloud/ecp/resources/global/regions/v1"
)

const (
	// RegionResource is the resource name used in API response metadata.
	RegionResource = rdom.Resource
	// RegionProviderName is the provider identifier used in response metadata.
	RegionProviderName = "secapi.cloud/v1"
	// RegionBaseURL is the base URL prefix for region refs.
	RegionBaseURL = "/providers/secapi.cloud"
)

// ListParamsFromAPI converts SDK ListRegionsParams to resource.ListParams.
func ListParamsFromAPI(params regionv1sdk.ListRegionsParams) resource.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return resource.ListParams{
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// DomainToAPIIterator converts a list of Region domain objects to an SDK RegionIterator.
func DomainToAPIIterator(domains []*rdom.Region, nextSkipToken *string) *regionv1sdk.RegionIterator {
	items := make([]sdkschema.Region, len(domains))
	for i, dom := range domains {
		items[i] = domainToAPI(*dom, "list")
	}

	iterator := &regionv1sdk.RegionIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: RegionProviderName,
			Resource: RegionResource,
			Verb:     "list",
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// DomainToAPI converts a Region domain object to a schema.Region for Get operations.
func DomainToAPI(domain *rdom.Region) sdkschema.Region {
	return domainToAPI(*domain, "get")
}

// domainToAPI converts a Region domain object to a schema.Region with the given verb.
func domainToAPI(dom rdom.Region, verb string) sdkschema.Region {
	providers := make([]sdkschema.Provider, 0, len(dom.Providers))
	for _, p := range dom.Providers {
		providers = append(providers, sdkschema.Provider{Name: p.Name, Url: p.URL, Version: p.Version})
	}
	zones := make([]sdkschema.Zone, 0, len(dom.Zones))
	for _, z := range dom.Zones {
		zones = append(zones, sdkschema.Zone(z))
	}

	resVersion := int64(0)
	if rv, err := strconv.ParseInt(dom.ResourceVersion, 10, 64); err == nil {
		resVersion = rv
	}

	sdk := sdkschema.Region{
		Spec: sdkschema.RegionSpec{
			AvailableZones: zones,
			Providers:      providers,
		},
		Metadata: &sdkschema.GlobalResourceMetadata{
			ApiVersion:      rdom.Version,
			CreatedAt:       dom.CreatedAt,
			LastModifiedAt:  dom.UpdatedAt,
			Kind:            sdkschema.GlobalResourceMetadataKindResourceKindRegion,
			Name:            dom.Name,
			Provider:        RegionProviderName,
			Resource:        dom.Name,
			Ref:             fmt.Sprintf("%s/%s", RegionBaseURL, dom.Name),
			ResourceVersion: resVersion,
			Verb:            verb,
		},
	}
	if dom.DeletedAt != nil {
		sdk.Metadata.DeletedAt = dom.DeletedAt
	}
	return sdk
}
