package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
)

// MapCRToRegionDomain converts either a concrete *Region or *unstructured.Unstructured into a *rdom.Region.
func MapCRToRegionDomain(obj client.Object) (*rdom.Region, error) {
	var cr Region

	switch t := obj.(type) {
	case *Region:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Region: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T (expected *Region or *unstructured.Unstructured)", obj)
	}

	if err := validateRegionSpec(cr); err != nil {
		return &rdom.Region{}, err
	}

	providers := mapProviders(cr)
	zones := mapZones(cr)

	meta := commondomain.Metadata{
		CommonMetadata: commondomain.CommonMetadata{
			Name:            cr.GetName(),
			Provider:        k8slabels.GetInternalLabels(cr.GetLabels())[k8slabels.InternalProviderLabel],
			ResourceVersion: cr.GetResourceVersion(),
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	return &rdom.Region{Metadata: meta, Providers: providers, Zones: zones}, nil
}

// MapRegionDomainToCR converts a *rdom.Region to a Kubernetes Region CR.
// Regions are read-only resources managed by the platform, so this primarily
// handles re-serialisation for update paths.
func MapRegionDomainToCR(d *rdom.Region) (client.Object, error) {
	if d == nil {
		return nil, fmt.Errorf("domain region is nil")
	}

	cr := &Region{}
	cr.SetName(d.Name)
	cr.SetResourceVersion(d.ResourceVersion)
	cr.SetGroupVersionKind(RegionGVK)

	// Spec fields are populated by the platform — return minimal CR for round-trip.
	// TODO: populate cr.Spec from d.Providers and d.Zones when schemav1.RegionSpec is available.

	return cr, nil
}

func validateRegionSpec(cr Region) error {
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

func mapProviders(cr Region) []rdom.Provider {
	providers := make([]rdom.Provider, 0, len(cr.Spec.Providers))
	for _, p := range cr.Spec.Providers {
		providers = append(providers, rdom.Provider{Name: p.Name, URL: p.Url, Version: p.Version})
	}
	return providers
}

func mapZones(cr Region) []rdom.Zone {
	zones := make([]rdom.Zone, 0, len(cr.Spec.AvailableZones))
	for _, z := range cr.Spec.AvailableZones {
		zones = append(zones, rdom.Zone(z))
	}
	return zones
}
