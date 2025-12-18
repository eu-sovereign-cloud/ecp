package kubernetes

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	regionsv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regions/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

// MapCRRegionToDomain converts either concrete *regionsv1.Region or *unstructured.Unstructured into a RegionDomain.
func MapCRRegionToDomain(obj client.Object) (*model.RegionDomain, error) {
	var cr regionsv1.Region
	switch t := obj.(type) {
	case *regionsv1.Region:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("convert unstructured to Region: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T (expected *regionsv1.Region or *unstructured.Unstructured)", obj)
	}

	if err := validateRegionSpec(cr); err != nil {
		return &model.RegionDomain{}, err
	}

	providers := mapProviders(cr)
	zones := mapZones(cr)

	meta := model.Metadata{
		Name:            cr.GetName(),
		Namespace:       cr.GetNamespace(),
		Labels:          cr.GetLabels(),
		ResourceVersion: cr.GetResourceVersion(),
		CreatedAt:       cr.GetCreationTimestamp().Time,
		UpdatedAt:       cr.GetCreationTimestamp().Time,
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	return &model.RegionDomain{Metadata: meta, Providers: providers, Zones: zones}, nil
}

func validateRegionSpec(cr regionsv1.Region) error {
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

func mapProviders(cr regionsv1.Region) []model.Provider {
	providers := make([]model.Provider, 0, len(cr.Spec.Providers))
	for _, p := range cr.Spec.Providers {
		providers = append(providers, model.Provider{Name: p.Name, URL: p.Url, Version: p.Version})
	}
	return providers
}

func mapZones(cr regionsv1.Region) []model.Zone {
	zones := make([]model.Zone, 0, len(cr.Spec.AvailableZones))
	for _, z := range cr.Spec.AvailableZones {
		zones = append(zones, model.Zone(z))
	}
	return zones
}

// MapRegionDomainToCR converts a RegionDomain into a concrete *regionsv1.Region.
func MapRegionDomainToCR(domain *model.RegionDomain) (client.Object, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain cannot be nil")
	}

	providers := make([]genv1.Provider, 0, len(domain.Providers))
	for _, p := range domain.Providers {
		providers = append(providers, genv1.Provider{
			Name:    p.Name,
			Url:     p.URL,
			Version: p.Version,
		})
	}

	zones := make([]string, 0, len(domain.Zones))
	for _, z := range domain.Zones {
		zones = append(zones, string(z))
	}

	cr := &regionsv1.Region{
		ObjectMeta: metav1.ObjectMeta{
			Name:      domain.Metadata.Name,
			Namespace: domain.Metadata.Namespace,
		},
		Spec: genv1.RegionSpec{
			Providers:      providers,
			AvailableZones: zones,
		},
	}

	if domain.Metadata.Labels != nil {
		cr.Labels = domain.Metadata.Labels
	}
	if domain.Metadata.ResourceVersion != "" {
		cr.ResourceVersion = domain.Metadata.ResourceVersion
	}

	return cr, nil
}

// RegionDomainToK8sConverter is a DomainToK8s converter for RegionDomain.
func RegionDomainToK8sConverter(domain *model.RegionDomain) (client.Object, error) {
	return MapRegionDomainToCR(domain)
}
