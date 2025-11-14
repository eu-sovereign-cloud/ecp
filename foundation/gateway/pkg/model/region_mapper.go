package model

import (
	"fmt"
	"strconv"

	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
)

const (
	RegionBaseURL      = "/providers/seca.region"
	ProviderRegionName = "seca.region/v1"
)

// MapRegionDomainToSDK converts a RegionDomain to an SDK schema.Region, embedding metadata verb.
func MapRegionDomainToSDK(dom RegionDomain, verb string) schema.Region {
	providers := make([]schema.Provider, 0, len(dom.Providers))
	for _, p := range dom.Providers {
		providers = append(providers, schema.Provider{Name: p.Name, Url: p.URL, Version: p.Version})
	}
	zones := make([]schema.Zone, 0, len(dom.Zones))
	for _, z := range dom.Zones {
		zones = append(zones, schema.Zone(z))
	}
	resVersion := 0
	// resourceVersion is best-effort numeric
	if rv, err := strconv.Atoi(dom.ResourceVersion); err == nil {
		resVersion = rv
	}
	refObj := schema.ReferenceObject{Resource: fmt.Sprintf("%s/%s", RegionBaseURL, dom.Name)}
	ref := schema.Reference{}
	_ = ref.FromReferenceObject(refObj) // ignore mapping error, not critical internally
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
			Provider:        ProviderRegionName,
			Resource:        dom.Name,
			Ref:             &ref,
			ResourceVersion: resVersion,
			Verb:            verb,
		},
	}
	if dom.DeletedAt != nil {
		sdk.Metadata.DeletedAt = dom.DeletedAt
	}
	return sdk
}
