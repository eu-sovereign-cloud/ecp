package kubernetes

import (
	"fmt"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"
	storageskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/skus/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func MapCRToNetworkSKUDomain(cr netowrkskuv1.SKU) *regional.NetworkSKUDomain {
	return &regional.NetworkSKUDomain{
		Metadata: model.Metadata{Name: cr.Name, Namespace: cr.Namespace},
		Spec: regional.NetworkSKUSpec{
			Bandwidth: cr.Spec.Bandwidth,
			Packets:   cr.Spec.Packets,
		},
	}
}

// MapCRToStorageSKUDomain converts either concrete *storageskuv1.StorageSKU or unstructured.Unstructured into a StorageSKUDomain.
func MapCRToStorageSKUDomain(obj client.Object) (*regional.StorageSKUDomain, error) {
	var cr storageskuv1.SKU

	switch t := obj.(type) {
	case *storageskuv1.SKU:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to StorageSKU: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

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

	return &regional.StorageSKUDomain{
		Metadata: meta,
		Spec: regional.StorageSKUSpec{
			Iops:          int64(cr.Spec.Iops),
			MinVolumeSize: int64(cr.Spec.MinVolumeSize),
			Type:          string(cr.Spec.Type),
		},
	}, nil
}

// MapStorageSKUDomainToCR converts a StorageSKUDomain into a concrete *storageskuv1.SKU.
func MapStorageSKUDomainToCR(domain *regional.StorageSKUDomain) (client.Object, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain cannot be nil")
	}

	cr := &storageskuv1.SKU{
		ObjectMeta: metav1.ObjectMeta{
			Name:      domain.Name,
			Namespace: domain.Namespace,
		},
		Spec: genv1.StorageSkuSpec{
			Iops:          int(domain.Spec.Iops),
			MinVolumeSize: int(domain.Spec.MinVolumeSize),
			Type:          genv1.StorageSkuSpecType(domain.Spec.Type),
		},
	}

	if domain.Labels != nil {
		cr.Labels = domain.Labels
	}
	if domain.ResourceVersion != "" {
		cr.ResourceVersion = domain.ResourceVersion
	}

	return cr, nil
}

// StorageSKUDomainToK8sConverter is a DomainToK8s converter for StorageSKUDomain.
func StorageSKUDomainToK8sConverter(domain *regional.StorageSKUDomain) (client.Object, error) {
	return MapStorageSKUDomainToCR(domain)
}
