package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network-sku"
)

// NetworkSKUFromCR converts either a concrete *NetworkSKU or *unstructured.Unstructured
// into a *skudom.NetworkSKU.
func NetworkSKUFromCR(obj client.Object) (*skudom.NetworkSKU, error) {
	var cr NetworkSKU

	switch t := obj.(type) {
	case *NetworkSKU:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to NetworkSKU: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)

	sku := &skudom.NetworkSKU{
		Spec: skudom.NetworkSKUSpec{
			Bandwidth: cr.Spec.Bandwidth,
			Packets:   cr.Spec.Packets,
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

// NetworkSKUToCR converts a *skudom.NetworkSKU to a Kubernetes NetworkSKU CR.
// NetworkSKUs are read-only resources — this is provided for completeness.
func NetworkSKUToCR(sku *skudom.NetworkSKU) (client.Object, error) {
	if sku == nil {
		return nil, fmt.Errorf("network SKU is nil")
	}

	cr := &NetworkSKU{}
	cr.SetName(sku.Name)
	cr.SetResourceVersion(sku.ResourceVersion)
	cr.SetGroupVersionKind(NetworkSKUGVK)

	// TODO: populate cr.Spec from sku.Spec when schemav1.NetworkSkuSpec fields are available

	return cr, nil
}
