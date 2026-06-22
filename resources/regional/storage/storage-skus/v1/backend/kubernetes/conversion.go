package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/labels"
	ssdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/storage-skus/v1/domain"
)

// MapCRToStorageSKUDomain converts either a concrete *StorageSKU or *unstructured.Unstructured
// into a *ssdom.StorageSKUDomain.
func MapCRToStorageSKUDomain(obj client.Object) (*ssdom.StorageSKUDomain, error) {
	var cr StorageSKU

	switch t := obj.(type) {
	case *StorageSKU:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to StorageSKU: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)

	sku := &ssdom.StorageSKUDomain{
		Spec: ssdom.StorageSKUSpecDomain{
			Iops:          int64(cr.Spec.Iops),
			MinVolumeSize: int64(cr.Spec.MinVolumeSize),
			Type:          string(cr.Spec.Type),
		},
	}
	sku.Name = cr.GetName()
	sku.ResourceVersion = cr.GetResourceVersion()
	sku.CreatedAt = cr.GetCreationTimestamp().Time
	sku.UpdatedAt = cr.GetCreationTimestamp().Time
	sku.Provider = internalLabels[k8slabels.InternalProviderLabel]
	sku.Region = internalLabels[k8slabels.InternalRegionLabel]
	sku.Tenant = internalLabels[k8slabels.InternalTenantLabel]

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		sku.DeletedAt = &ts.Time
	}

	return sku, nil
}

// MapStorageSKUDomainToCR converts a *ssdom.StorageSKUDomain to a Kubernetes StorageSKU CR.
// StorageSKUs are read-only resources — this is provided for completeness.
func MapStorageSKUDomainToCR(d *ssdom.StorageSKUDomain) (client.Object, error) {
	if d == nil {
		return nil, fmt.Errorf("domain storage SKU is nil")
	}

	cr := &StorageSKU{}
	cr.SetName(d.Name)
	cr.SetResourceVersion(d.ResourceVersion)
	cr.SetGroupVersionKind(StorageSKUGVK)

	// TODO: populate cr.Spec from d.Spec when genv1.StorageSkuSpec fields are available

	return cr, nil
}
