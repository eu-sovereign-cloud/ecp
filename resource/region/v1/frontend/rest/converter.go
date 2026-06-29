// Package rest provides REST↔domain conversion and HTTP handlers for the region resource.
package rest

import (
	"fmt"
	"strconv"

	regionv1sdk "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"
)

const (
	// RegionResource is the resource name used in API response metadata.
	RegionResource = rdom.Resource
	// RegionProviderName is the provider identifier used in response metadata.
	RegionProviderName = "secapi.cloud/v1"
	// regionRefBaseURL is the base URL prefix for region refs in the secapi.cloud API.
	// Distinct from rdom.RegionBaseURL ("/providers/seca.region") which is the domain-layer identifier.
	regionRefBaseURL = "/providers/secapi.cloud"
)

// listParamsFromAPI converts SDK ListRegionsParams to resource.ListParams.
func listParamsFromAPI(params regionv1sdk.ListRegionsParams) resource.ListParams {
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

// regionIteratorToAPI converts a list of Region domain objects to an SDK RegionIterator.
func regionIteratorToAPI(rs []*rdom.Region, nextSkipToken *string) *regionv1sdk.RegionIterator {
	items := make([]sdkschema.Region, len(rs))
	for i, r := range rs {
		items[i] = regionToAPI(*r, "list")
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

// regionToAPIForGet converts a Region domain object to a schema.Region for Get operations.
func regionToAPIForGet(r *rdom.Region) sdkschema.Region {
	return regionToAPI(*r, "get")
}

// regionToAPI converts a Region domain object to a schema.Region with the given verb.
func regionToAPI(r rdom.Region, verb string) sdkschema.Region {
	providers := make([]sdkschema.Provider, 0, len(r.Providers))
	for _, p := range r.Providers {
		providers = append(providers, sdkschema.Provider{Name: p.Name, Url: p.URL, Version: p.Version})
	}
	zones := make([]sdkschema.Zone, 0, len(r.Zones))
	for _, z := range r.Zones {
		zones = append(zones, sdkschema.Zone(z))
	}

	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(r.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	sdk := sdkschema.Region{
		Spec: sdkschema.RegionSpec{
			AvailableZones: zones,
			Providers:      providers,
		},
		Metadata: &sdkschema.GlobalResourceMetadata{
			ApiVersion:      rdom.Version,
			CreatedAt:       r.CreatedAt,
			LastModifiedAt:  r.UpdatedAt,
			Kind:            sdkschema.GlobalResourceMetadataKindResourceKindRegion,
			Name:            r.Name,
			Provider:        RegionProviderName,
			Resource:        r.Name,
			Ref:             fmt.Sprintf("%s/%s", regionRefBaseURL, r.Name),
			ResourceVersion: resourceVersion,
			Verb:            verb,
		},
	}
	if r.DeletedAt != nil {
		sdk.Metadata.DeletedAt = r.DeletedAt
	}
	return sdk
}
