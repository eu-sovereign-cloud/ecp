package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	skudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
)

// InstanceSKUFromCR converts either a concrete *InstanceSKU or *unstructured.Unstructured
// into a *skudom.InstanceSKU.
func InstanceSKUFromCR(obj client.Object) (*skudom.InstanceSKU, error) {
	var cr InstanceSKU

	switch t := obj.(type) {
	case *InstanceSKU:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to InstanceSKU: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)

	sku := &skudom.InstanceSKU{
		Spec: skudom.InstanceSKUSpec{
			VCPU: cr.Spec.VCPU,
			Ram:  cr.Spec.Ram,
		},
	}
	sku.Name = cr.GetName()
	sku.ResourceVersion = cr.GetResourceVersion()
	sku.CreatedAt = cr.GetCreationTimestamp().Time
	sku.UpdatedAt = cr.GetCreationTimestamp().Time
	sku.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	sku.Region = internalLabels[k8slabels.InternalRegionLabel]
	sku.Tenant = internalLabels[k8slabels.InternalTenantLabel]

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		sku.DeletedAt = &ts.Time
	}

	return sku, nil
}

// InstanceSKUToCR converts a *skudom.InstanceSKU to a Kubernetes InstanceSKU CR.
// InstanceSKUs are read-only resources — this is provided for completeness.
func InstanceSKUToCR(sku *skudom.InstanceSKU) (client.Object, error) {
	if sku == nil {
		return nil, fmt.Errorf("instance SKU is nil")
	}

	cr := &InstanceSKU{
		Spec: InstanceSkuSpec{
			VCPU: sku.Spec.VCPU,
			Ram:  sku.Spec.Ram,
		},
	}
	cr.SetName(sku.Name)
	cr.SetResourceVersion(sku.ResourceVersion)
	cr.SetGroupVersionKind(InstanceSKUGVK)

	return cr, nil
}
