package kubernetes

import (
	"fmt"

	v1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// MapCRRegionToDomain converts either concrete *v1.Region or *unstructured.Unstructured into a RegionDomain.
func MapCRRegionToDomain(obj client.Object) (*model.RegionDomain, error) {
	var cr v1.Region
	switch t := obj.(type) {
	case *v1.Region:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("convert unstructured to Region: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T (expected *v1.Region or *unstructured.Unstructured)", obj)
	}

	if err := validateRegionSpec(cr); err != nil {
		return &model.RegionDomain{}, err
	}

	providers := mapProviders(cr)
	zones := mapZones(cr)

	meta := model.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			Provider:        labels.GetInternalLabels(cr.GetLabels())[labels.InternalProviderLabel],
			ResourceVersion: cr.GetResourceVersion(),
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	return &model.RegionDomain{Metadata: meta, Providers: providers, Zones: zones}, nil
}

func validateRegionSpec(cr v1.Region) error {
	if len(cr.Spec.Providers) == 0 {
		return fmt.Errorf("region %s has no providers", cr.Name)
	}
	if len(cr.Spec.AvailableZones) == 0 {
		return fmt.Errorf("region %s has no available zones", cr.Name)
	}
	for _, p := range cr.Spec.Providers {
		if p.Name == "" {
			return fmt.Errorf("region %s has provider with empty name", cr.Name)
		}
	}
	for _, z := range cr.Spec.AvailableZones {
		if z == "" {
			return fmt.Errorf("region %s has empty zone entry", cr.Name)
		}
	}
	return nil
}

func mapProviders(cr v1.Region) []model.Provider {
	providers := make([]model.Provider, 0, len(cr.Spec.Providers))
	for _, p := range cr.Spec.Providers {
		providers = append(providers, model.Provider{Name: p.Name, URL: p.Url, Version: p.Version})
	}
	return providers
}

func mapZones(cr v1.Region) []model.Zone {
	zones := make([]model.Zone, 0, len(cr.Spec.AvailableZones))
	for _, z := range cr.Spec.AvailableZones {
		zones = append(zones, model.Zone(z))
	}
	return zones
}
