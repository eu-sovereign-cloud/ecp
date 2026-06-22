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

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/labels"
	genv1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resources/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resources/regional/storage/block-storages/v1/domain"
)

// MapCRToBlockStorageDomain converts either a concrete *BlockStorage or *unstructured.Unstructured
// into a *bsdom.BlockStorageDomain.
func MapCRToBlockStorageDomain(obj client.Object) (*bsdom.BlockStorageDomain, error) {
	var cr BlockStorage

	switch t := obj.(type) {
	case *BlockStorage:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to BlockStorage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	spec := bsdom.BlockStorageSpecDomain{
		SizeGB: cr.Spec.SizeGB,
		SkuRef: commonbackend.MapCRToReferenceDomain(cr.Spec.SkuRef),
	}
	if cr.Spec.SourceImageRef != nil {
		ref := commonbackend.MapCRToReferenceDomain(*cr.Spec.SourceImageRef)
		spec.SourceImageRef = &ref
	}

	bs := &bsdom.BlockStorageDomain{
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

	bs.Status = &bsdom.BlockStorageStatusDomain{}
	if cr.Status != nil {
		bs.Status = &bsdom.BlockStorageStatusDomain{
			SizeGB: cr.Status.SizeGB,
		}
		bs.Status.State = commonbackend.MapCRToResourceStateDomain(cr.Status.State)
		bs.Status.Conditions = commonbackend.MapCRToStatusConditionDomains(cr.Status.Conditions)
		if cr.Status.AttachedTo != nil {
			ref := commonbackend.MapCRToReferenceDomain(*cr.Status.AttachedTo)
			bs.Status.AttachedTo = &ref
		}
	} else {
		bs.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return bs, nil
}

// MapBlockStorageDomainToCR converts a *bsdom.BlockStorageDomain to a Kubernetes BlockStorage CR.
func MapBlockStorageDomainToCR(d *bsdom.BlockStorageDomain) (client.Object, error) {
	if d == nil {
		return nil, fmt.Errorf("domain block storage is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(d.Labels)
	crLabels[k8slabels.InternalTenantLabel] = d.Tenant
	crLabels[k8slabels.InternalWorkspaceLabel] = d.Workspace
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(d.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = d.Region

	cr := &BlockStorage{
		ObjectMeta: v1.ObjectMeta{
			Name:            d.Name,
			Namespace:       k8sadapter.ComputeNamespace(d),
			Labels:          crLabels,
			ResourceVersion: d.ResourceVersion,
		},
		CommonData: genv1.CommonData{
			Annotations: d.Annotations,
			Extensions:  d.Extensions,
			Labels:      slices.Collect(maps.Keys(d.Labels)),
		},
		Spec: BlockStorageSpec{
			SizeGB: d.Spec.SizeGB,
			SkuRef: commonbackend.MapReferenceDomainToCR(d.Spec.SkuRef),
		},
	}
	cr.SetGroupVersionKind(BlockStorageGVK)

	if d.Spec.SourceImageRef != nil {
		ref := commonbackend.MapReferenceDomainToCR(*d.Spec.SourceImageRef)
		cr.Spec.SourceImageRef = &ref
	}

	if d.Status != nil && len(d.Status.Conditions) > 0 {
		state := commonbackend.MapResourceStateDomainToCR(d.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &BlockStorageStatus{
			SizeGB:     d.Status.SizeGB,
			Conditions: commonbackend.MapStatusConditionDomainsToCR(d.Status.Conditions),
			State:      *state,
		}
		if d.Status.AttachedTo != nil {
			ref := commonbackend.MapReferenceDomainToCR(*d.Status.AttachedTo)
			cr.Status.AttachedTo = &ref
		}
	}

	return cr, nil
}
