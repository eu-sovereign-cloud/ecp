package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageinstancev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/instances/v1"
	storageskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/skus/v1"
	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func MapCRToNetworkSKUDomain(cr netowrkskuv1.NetworkSKU) *regional.NetworkSKUDomain {
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
	var cr storageskuv1.StorageSKU

	switch t := obj.(type) {
	case *storageskuv1.StorageSKU:
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

// MapCRToBlockStorageDomain converts either concrete *storageinstancev1.Storage or unstructured.Unstructured into a BlockStorageDomain.
func MapCRToBlockStorageDomain(obj client.Object) (*regional.BlockStorageDomain, error) {
	var cr storageinstancev1.Storage

	switch t := obj.(type) {
	case *storageinstancev1.Storage:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to K8sConverter unstructured to Storage: %w", err)
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

	domain := &regional.BlockStorageDomain{
		Metadata: meta,
		Spec: regional.BlockStorageSpec{
			SizeGB:         cr.Spec.SizeGB,
			SkuRef:         cr.Spec.SkuRef,
			SourceImageRef: cr.Spec.SourceImageRef,
		},
	}

	if cr.Status.State != nil || len(cr.Status.Conditions) > 0 {
		domain.Status = &regional.BlockStorageStatus{
			SizeGB:     cr.Status.SizeGB,
			Conditions: cr.Status.Conditions,
			AttachedTo: cr.Status.AttachedTo,
			State:      cr.Status.State,
		}
	}

	return domain, nil
}

// MapBlockStorageDomainToCR converts a BlockStorageDomain to a Storage CR.
func MapBlockStorageDomainToCR(domain *regional.BlockStorageDomain) (*unstructured.Unstructured, error) {
	cr := &storageinstancev1.Storage{}
	cr.SetName(domain.GetName())
	cr.SetNamespace(domain.GetNamespace())
	cr.SetLabels(domain.Labels)

	// Spec and Status use genv1 types directly
	cr.Spec.SizeGB = domain.Spec.SizeGB
	cr.Spec.SkuRef = domain.Spec.SkuRef
	cr.Spec.SourceImageRef = domain.Spec.SourceImageRef
	if domain.Status != nil {
		cr.Status.SizeGB = domain.Status.SizeGB
		cr.Status.Conditions = domain.Status.Conditions
		cr.Status.AttachedTo = domain.Status.AttachedTo
		cr.Status.State = domain.Status.State
	}

	// Convert to unstructured for K8s API calls
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to K8sConverter Storage CR to unstructured: %w", err)
	}

	u := &unstructured.Unstructured{Object: unstructuredObj}
	u.SetGroupVersionKind(storageinstancev1.StorageGVK)

	return u, nil
}
