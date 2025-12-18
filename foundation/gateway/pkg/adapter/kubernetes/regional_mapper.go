package kubernetes

import (
	"fmt"
	"maps"
	"slices"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"
	storageskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/skus/v1"
	workspacev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/convert"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func MapCRToNetworkSKUDomain(cr netowrkskuv1.SKU) *regional.NetworkSKUDomain {
	return &regional.NetworkSKUDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: cr.GetName(),
			},
		},
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

	internalLabels := labels.GetInternalLabels(cr.GetLabels())
	meta := regional.Metadata{
		Labels: labels.GetCSPLabels(cr.GetLabels()),
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			Provider:        internalLabels[labels.InternalProviderLabel],
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
		Region: internalLabels[labels.InternalRegionLabel],
		Tenant: internalLabels[labels.InternalTenantLabel],
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

// MapCRToWorkspaceDomain converts either concrete *workspacev1.Workspace or unstructured.Unstructured into a *regional.WorkspaceDomain.
func MapCRToWorkspaceDomain(obj client.Object) (*regional.WorkspaceDomain, error) {
	var cr workspacev1.Workspace

	switch t := obj.(type) {
	case *workspacev1.Workspace:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Workspace: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	spec := make(map[string]interface{}, len(cr.Spec))
	for k, v := range cr.Spec {
		spec[k] = v
	}

	internalLabels := labels.GetInternalLabels(cr.GetLabels())
	keyedLabels := labels.GetKeyedLabels(cr.GetLabels())
	// NOTE: Do we expect CSP labels on resources created by a user? If so, they'll need to be added as well.

	meta := regional.Metadata{
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			CreatedAt:       cr.GetCreationTimestamp().Time,
			Provider:        internalLabels[labels.InternalProviderLabel],
		},
		Region:      internalLabels[labels.InternalRegionLabel],
		Tenant:      internalLabels[labels.InternalTenantLabel],
		Labels:      labels.ComputeOriginalFromKeyedLabels(keyedLabels, cr.RegionalCommonData.Labels),
		Annotations: cr.RegionalCommonData.Annotations,
		Extensions:  cr.RegionalCommonData.Extensions,
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	status := regional.WorkspaceStatusDomain{
		StatusDomain: regional.StatusDomain{
			State:      (*string)(cr.Status.State),
			Conditions: mapCRToStatusConditionDomains(cr.Status.Conditions),
		},
		ResourceCount: cr.Status.ResourceCount,
	}

	return &regional.WorkspaceDomain{
		Metadata: meta,
		Spec:     spec,
		Status:   status,
	}, nil
}

// MapWorkspaceDomainToCR maps a WorkspaceDomain to a Workspace CR.
func MapWorkspaceDomainToCR(domain *regional.WorkspaceDomain) (client.Object, error) {
	if domain == nil {
		return nil, fmt.Errorf("domain workspace is nil")
	}

	spec := make(map[string]string, len(domain.Spec))
	for k, v := range domain.Spec {
		spec[k] = convert.InterfaceToString(v)
	}

	crLabels := labels.ComputeKeyedFromOriginalLabels(domain.Metadata.Labels)
	crLabels[labels.InternalTenantLabel] = domain.Metadata.Tenant

	return &workspacev1.Workspace{
		ObjectMeta: v1.ObjectMeta{
			Name:      domain.Metadata.Name,
			Namespace: computeNamespace(domain),
			Labels:    crLabels,
		},
		RegionalCommonData: common.RegionalCommonData{
			Annotations: domain.Metadata.Annotations,
			Extensions:  domain.Metadata.Extensions,
			Labels:      slices.Collect(maps.Keys(domain.Metadata.Labels)),
		},
		Spec: spec,
	}, nil
}

// mapCRToStatusConditionDomain maps a types.StatusCondition to a regional.StatusConditionDomain.
func mapCRToStatusConditionDomain(crStatusCondition types.StatusCondition) regional.StatusConditionDomain {
	return regional.StatusConditionDomain{
		Type:             crStatusCondition.Type,
		State:            string(crStatusCondition.State),
		LastTransitionAt: crStatusCondition.LastTransitionAt.Time,
		Reason:           crStatusCondition.Reason,
		Message:          crStatusCondition.Message,
	}
}

// mapCRToStatusConditionDomains maps a slice of types.StatusCondition to a slice of regional.StatusConditionDomain.
func mapCRToStatusConditionDomains(crStatusConditions []types.StatusCondition) []regional.StatusConditionDomain {
	conditions := make([]regional.StatusConditionDomain, len(crStatusConditions))
	for i, cond := range crStatusConditions {
		conditions[i] = mapCRToStatusConditionDomain(cond)
	}
	return conditions
}
