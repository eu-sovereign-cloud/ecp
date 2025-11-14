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

// MapRegionCRDToDomain converts a Kubernetes Region CRD to a RegionDomain enforcing invariants.
func MapRegionCRDToDomain(cr regionsv1.Region) (*RegionDomain, error) {
	var dom RegionDomain
	if len(cr.Spec.Providers) == 0 {
		return &dom, fmt.Errorf("region %s has no providers", cr.Name)
	}
	if len(cr.Spec.AvailableZones) == 0 {
		return &dom, fmt.Errorf("region %s has no available zones", cr.Name)
	}
	providers := make([]Provider, 0, len(cr.Spec.Providers))
	for _, p := range cr.Spec.Providers {
		if p.Name == "" {
			return &dom, fmt.Errorf("region %s has provider with empty name", cr.Name)
		}
		providers = append(providers, Provider{Name: p.Name, URL: p.Url, Version: p.Version})
	}
	zones := make([]string, 0, len(cr.Spec.AvailableZones))
	for _, z := range cr.Spec.AvailableZones {
		if z == "" {
			return &dom, fmt.Errorf("region %s has empty zone entry", cr.Name)
		}
		zones = append(zones, z)
	}
	meta := Metadata{
		Name:            cr.GetName(),
		Labels:          cr.GetLabels(),
		ResourceVersion: cr.GetResourceVersion(),
		CreatedAt:       cr.GetCreationTimestamp().Time,
		UpdatedAt:       cr.GetCreationTimestamp().Time,
	}
	if cr.GetDeletionTimestamp() != nil {
		meta.DeletedAt = &cr.GetDeletionTimestamp().Time
	}
	dom = RegionDomain{Metadata: Metadata{Namespace: meta.Namespace, Name: meta.Name}, Providers: providers, Zones: zones}
	return &dom, nil
}

// MapRegionDomainToSDK converts a RegionDomain to an SDK schema.Region, embedding metadata verb.
func MapRegionDomainToSDK(dom RegionDomain, verb string) schema.Region {
	providers := make([]schema.Provider, 0, len(dom.Providers))
	for _, p := range dom.Providers {
		providers = append(providers, schema.Provider{Name: p.Name, Url: p.URL, Version: p.Version})
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
			AvailableZones: dom.Zones,
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
