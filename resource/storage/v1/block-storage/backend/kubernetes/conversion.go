package kubernetes

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

// BlockStorageFromCR converts either a concrete *BlockStorage or *unstructured.Unstructured
// into a *bsdom.BlockStorage.
func BlockStorageFromCR(obj client.Object) (*bsdom.BlockStorage, error) {
	var cr BlockStorage

	switch t := obj.(type) {
	case *BlockStorage:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, kernel.NewError(kernel.KindValidation, fmt.Errorf("failed to convert unstructured to BlockStorage: %w", err))
		}
	default:
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("unsupported object type %T", obj))
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := bsdom.BlockStorageSpec{
		SizeGB: cr.Spec.SizeGB,
		SkuRef: commonbackend.ReferenceFromCR(cr.Spec.SkuRef),
	}
	if cr.Spec.SourceImageRef != nil {
		ref := commonbackend.ReferenceFromCR(*cr.Spec.SourceImageRef)
		spec.SourceImageRef = &ref
	}

	bs := &bsdom.BlockStorage{
		Spec: spec,
	}
	bs.Name = cr.GetName()
	bs.ResourceVersion = cr.GetResourceVersion()
	bs.CreatedAt = cr.GetCreationTimestamp().Time
	bs.UpdatedAt = cr.GetCreationTimestamp().Time
	bs.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	bs.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	bs.Workspace = internalLabels[k8slabels.InternalWorkspaceLabel]
	bs.Region = internalLabels[k8slabels.InternalRegionLabel]
	bs.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	bs.Annotations = cr.CommonData.Annotations
	bs.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		bs.DeletedAt = &ts.Time
	}

	bs.Status = &bsdom.BlockStorageStatus{}
	if cr.Status != nil {
		bs.Status = &bsdom.BlockStorageStatus{
			SizeGB: cr.Status.SizeGB,
		}
		status, err := commonbackend.StatusFromCR(cr.Status.State, cr.Status.Conditions)
		if err != nil {
			return nil, fmt.Errorf("block storage %s: %w", cr.Name, err)
		}
		bs.Status.Status = status
		if cr.Status.AttachedTo != nil {
			ref := commonbackend.ReferenceFromCR(*cr.Status.AttachedTo)
			bs.Status.AttachedTo = &ref
		}
	} else {
		bs.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return bs, nil
}

// BlockStorageToCR converts a *bsdom.BlockStorage to a Kubernetes BlockStorage CR.
func BlockStorageToCR(bs *bsdom.BlockStorage) (client.Object, error) {
	if bs == nil {
		return nil, kernel.NewError(kernel.KindInternal, fmt.Errorf("block storage is nil"))
	}

	crLabels := k8slabels.OriginalToKeyed(bs.Labels)
	crLabels[k8slabels.InternalTenantLabel] = bs.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = bs.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(bs.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = bs.Region

	cr := &BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:            bs.Name,
			Namespace:       k8sadapter.ComputeNamespace(bs),
			Labels:          crLabels,
			ResourceVersion: bs.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: bs.Annotations,
			Extensions:  bs.Extensions,
			Labels:      slices.Sorted(maps.Keys(bs.Labels)),
		},
		Spec: BlockStorageSpec{
			SizeGB: bs.Spec.SizeGB,
			SkuRef: commonbackend.ReferenceToCR(bs.Spec.SkuRef),
		},
	}
	cr.SetGroupVersionKind(BlockStorageGVK)

	if bs.Spec.SourceImageRef != nil {
		ref := commonbackend.ReferenceToCR(*bs.Spec.SourceImageRef)
		cr.Spec.SourceImageRef = &ref
	}

	if bs.Status != nil && len(bs.Status.Conditions) > 0 {
		state, conds, err := commonbackend.StatusToCR(bs.Status.Status)
		if err != nil {
			return nil, fmt.Errorf("block storage %s: %w", bs.Name, err)
		}
		cr.Status = &BlockStorageStatus{
			SizeGB:     bs.Status.SizeGB,
			Conditions: conds,
			State:      state,
		}
		if bs.Status.AttachedTo != nil {
			ref := commonbackend.ReferenceToCR(*bs.Status.AttachedTo)
			cr.Status.AttachedTo = &ref
		}
	}

	return cr, nil
}

// Converter is the CR<->domain conversion pair for BlockStorage.
var Converter = k8sadapter.TwoWayConverter[*bsdom.BlockStorage]{
	FromCR: BlockStorageFromCR,
	ToCR:   BlockStorageToCR,
}
