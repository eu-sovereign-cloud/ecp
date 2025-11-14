package kubernetes

import (
	"fmt"

	v1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// MapRegionCRDToDomain converts a Kubernetes Region CRD to a RegionDomain enforcing invariants.
func MapRegionCRDToDomain(cr v1.Region) (*model.RegionDomain, error) {
	var dom model.RegionDomain
	if len(cr.Spec.Providers) == 0 {
		return &dom, fmt.Errorf("region %s has no providers", cr.Name)
	}
	if len(cr.Spec.AvailableZones) == 0 {
		return &dom, fmt.Errorf("region %s has no available zones", cr.Name)
	}
	providers := make([]model.Provider, 0, len(cr.Spec.Providers))
	for _, p := range cr.Spec.Providers {
		if p.Name == "" {
			return &dom, fmt.Errorf("region %s has provider with empty name", cr.Name)
		}
		providers = append(providers, model.Provider{Name: p.Name, URL: p.Url, Version: p.Version})
	}
	zones := make([]model.Zone, 0, len(cr.Spec.AvailableZones))
	for _, z := range cr.Spec.AvailableZones {
		if z == "" {
			return &dom, fmt.Errorf("region %s has empty zone entry", cr.Name)
		}
		zones = append(zones, model.Zone(z))
	}
	meta := model.Metadata{
		Name:            cr.GetName(),
		Labels:          cr.GetLabels(),
		ResourceVersion: cr.GetResourceVersion(),
		CreatedAt:       cr.GetCreationTimestamp().Time,
		UpdatedAt:       cr.GetCreationTimestamp().Time,
	}
	if cr.GetDeletionTimestamp() != nil {
		meta.DeletedAt = &cr.GetDeletionTimestamp().Time
	}
	dom = model.RegionDomain{Metadata: model.Metadata{Namespace: meta.Namespace, Name: meta.Name}, Providers: providers, Zones: zones}
	return &dom, nil
}
