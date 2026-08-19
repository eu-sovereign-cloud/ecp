package kubernetes

import (
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	rdom "github.com/eu-sovereign-cloud/ecp/resource/region/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
)

// RegionFromCR converts either a concrete *Region or *unstructured.Unstructured into a *rdom.Region.
func RegionFromCR(obj client.Object) (*rdom.Region, error) {
	var cr Region

	switch t := obj.(type) {
	case *Region:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert unstructured to Region: %w", err))
		}
	default:
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("unsupported object type %T (expected *Region or *unstructured.Unstructured)", obj))
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

// RegionToCR converts a *rdom.Region to a Kubernetes Region CR.
// Regions are read-only resources managed by the platform, so this primarily
// handles re-serialisation for update paths.
func RegionToCR(r *rdom.Region) (client.Object, error) {
	if r == nil {
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("region is nil"))
	}

	cr := &Region{
		Spec: RegionSpec{
			Providers:      make([]Provider, 0, len(r.Providers)),
			AvailableZones: make([]string, 0, len(r.Zones)),
		},
	}
	for _, p := range r.Providers {
		cr.Spec.Providers = append(cr.Spec.Providers, Provider{Name: p.Name, Url: p.URL, Version: p.Version})
	}
	for _, z := range r.Zones {
		cr.Spec.AvailableZones = append(cr.Spec.AvailableZones, string(z))
	}

	cr.SetName(r.Name)
	cr.SetResourceVersion(r.ResourceVersion)
	cr.SetGroupVersionKind(RegionGVK)

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
	if slices.Contains(cr.Spec.AvailableZones, "") {
		return fmt.Errorf("region %s has empty zone entry", cr.Name)
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

// Converter is the CR<->domain conversion pair for Region, so a call site names one value
// instead of pairing the two directions by hand. See doc/CONVENTIONS.md §2.
var Converter = k8sadapter.TwoWayConverter[*rdom.Region]{
	FromCR: RegionFromCR,
	ToCR:   RegionToCR,
}
