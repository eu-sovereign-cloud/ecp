package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"
	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network/skus/v1"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	storageskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/skus/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
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

	crLabels := cr.GetLabels()
	internalLabels := labels.GetInternalLabels(crLabels)
	meta := regional.Metadata{
		Labels: labels.GetCSPLabels(crLabels),
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			Provider:        internalLabels[labels.InternalProviderLabel],
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
		Region: internalLabels[labels.InternalRegionLabel],
		Scope: scope.Scope{
			Tenant: internalLabels[labels.InternalTenantLabel],
		},
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

// MapCRToBlockStorageDomain converts either concrete *blockstoragev1.BlockStorage or unstructured.Unstructured into a BlockStorageDomain.
func MapCRToBlockStorageDomain(obj client.Object) (*regional.BlockStorageDomain, error) {
	var cr blockstoragev1.BlockStorage

	switch t := obj.(type) {
	case *blockstoragev1.BlockStorage:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to BlockStorage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := labels.GetInternalLabels(crLabels)
	meta := regional.Metadata{
		Labels: labels.GetCSPLabels(crLabels),
		CommonMetadata: model.CommonMetadata{
			Name:            cr.GetName(),
			ResourceVersion: cr.GetResourceVersion(),
			Provider:        internalLabels[labels.InternalProviderLabel],
			CreatedAt:       cr.GetCreationTimestamp().Time,
			UpdatedAt:       cr.GetCreationTimestamp().Time,
		},
		Region: internalLabels[labels.InternalRegionLabel],
		Scope: scope.Scope{
			Tenant:    internalLabels[labels.InternalTenantLabel],
			Workspace: internalLabels[labels.InternalWorkspaceLabel],
		},
	}
	if ts := cr.GetDeletionTimestamp(); ts != nil {
		meta.DeletedAt = &ts.Time
	}

	domain := &regional.BlockStorageDomain{
		Metadata: meta,
		Spec: regional.BlockStorageSpec{
			SizeGB: cr.Spec.SizeGB,
			SkuRef: mapCRReferenceObjectToDomain(cr.Spec.SkuRef),
		},
	}

	if cr.Spec.SourceImageRef != nil {
		ref := mapCRReferenceObjectToDomain(*cr.Spec.SourceImageRef)
		domain.Spec.SourceImageRef = &ref
	}

	if cr.Status.State != nil {
		domain.Status = &regional.BlockStorageStatus{
			SizeGB:     cr.Status.SizeGB,
			Conditions: mapCRStatusConditionsToDomain(cr.Status.Conditions),
		}
		if cr.Status.AttachedTo != nil {
			ref := mapCRReferenceObjectToDomain(*cr.Status.AttachedTo)
			domain.Status.AttachedTo = &ref
		}
		if cr.Status.State != nil {
			state := regional.ResourceState(*cr.Status.State)
			domain.Status.State = &state
		}
	}

	return domain, nil
}

// mapCRReferenceObjectToDomain converts a generated types.ReferenceObject to a domain ReferenceObject.
func mapCRReferenceObjectToDomain(ref genv1.ReferenceObject) regional.ReferenceObject {
	return regional.ReferenceObject{
		Provider:  ptr.Deref(ref.Provider, ""),
		Region:    ptr.Deref(ref.Region, ""),
		Resource:  ref.Resource,
		Tenant:    ptr.Deref(ref.Tenant, ""),
		Workspace: ptr.Deref(ref.Workspace, ""),
	}
}

// mapCRStatusConditionsToDomain converts generated StatusConditions to domain StatusConditions.
func mapCRStatusConditionsToDomain(conditions []genv1.StatusCondition) []regional.StatusCondition {
	result := make([]regional.StatusCondition, len(conditions))
	for i, c := range conditions {
		result[i] = regional.StatusCondition{
			LastTransitionAt: c.LastTransitionAt.Time,
			Message:          ptr.Deref(c.Message, ""),
			Reason:           ptr.Deref(c.Reason, ""),
			State:            regional.ResourceState(c.State),
			Type:             ptr.Deref(c.Type, ""),
		}
	}
	return result
}

// MapBlockStorageDomainToCR converts a BlockStorageDomain to a Kubernetes BlockStorage CR.
func MapBlockStorageDomainToCR(domain *regional.BlockStorageDomain) (client.Object, error) {
	cr := &blockstoragev1.BlockStorage{}
	cr.SetGroupVersionKind(blockstoragev1.BlockStorageGVR.GroupVersion().WithKind("BlockStorage"))
	cr.SetName(domain.Name)

	// Merge CSP labels with internal labels
	allLabels := make(map[string]string)
	for k, v := range domain.Labels {
		allLabels[k] = v
	}
	allLabels[labels.InternalTenantLabel] = domain.Tenant
	allLabels[labels.InternalWorkspaceLabel] = domain.Workspace
	if domain.Region != "" {
		allLabels[labels.InternalRegionLabel] = domain.Region
	}
	if domain.Provider != "" {
		allLabels[labels.InternalProviderLabel] = domain.Provider
	}
	cr.SetLabels(allLabels)

	cr.Spec = genv1.BlockStorageSpec{
		SizeGB: domain.Spec.SizeGB,
		SkuRef: mapDomainReferenceObjectToCR(domain.Spec.SkuRef),
	}

	if domain.Spec.SourceImageRef != nil {
		ref := mapDomainReferenceObjectToCR(*domain.Spec.SourceImageRef)
		cr.Spec.SourceImageRef = &ref
	}

	return cr, nil
}

// mapDomainReferenceObjectToCR converts a domain ReferenceObject to a generated types ReferenceObject.
// It parses the Resource path to extract embedded segments (providers, regions, tenants, workspaces)
// and sets the corresponding fields. Extracted segments are stripped from the Resource path.
// If a segment is not in the path, it falls back to the domain value.
func mapDomainReferenceObjectToCR(ref regional.ReferenceObject) genv1.ReferenceObject {
	resource := ref.Resource
	result := genv1.ReferenceObject{}

	// Extract values from Resource path or fall back to domain values
	if provider, remaining := extractAndStripSegment(resource, "providers/"); provider != "" {
		result.Provider = ptr.To(provider)
		resource = remaining
	} else if ref.Provider != "" {
		result.Provider = ptr.To(ref.Provider)
	}

	if region, remaining := extractAndStripSegment(resource, "regions/"); region != "" {
		result.Region = ptr.To(region)
		resource = remaining
	} else if ref.Region != "" {
		result.Region = ptr.To(ref.Region)
	}

	if tenant, remaining := extractAndStripSegment(resource, "tenants/"); tenant != "" {
		result.Tenant = ptr.To(tenant)
		resource = remaining
	} else if ref.Tenant != "" {
		result.Tenant = ptr.To(ref.Tenant)
	}

	if workspace, remaining := extractAndStripSegment(resource, "workspaces/"); workspace != "" {
		result.Workspace = ptr.To(workspace)
		resource = remaining
	} else if ref.Workspace != "" {
		result.Workspace = ptr.To(ref.Workspace)
	}

	result.Resource = resource
	return result
}

// extractAndStripSegment extracts the value following a segment prefix in a resource path
// and returns the remaining path with the segment removed.
// For example, extractAndStripSegment("workspaces/ws-1/block-storages/my-storage", "workspaces/")
// returns ("ws-1", "block-storages/my-storage").
// Returns empty strings if the segment is not found.
func extractAndStripSegment(resource, segment string) (value, remaining string) {
	var startIdx int
	var prefixLen int

	if strings.HasPrefix(resource, segment) {
		startIdx = len(segment)
		prefixLen = 0
	} else if idx := strings.Index(resource, "/"+segment); idx >= 0 {
		startIdx = idx + 1 + len(segment)
		prefixLen = idx
	} else {
		return "", ""
	}

	// Find the end of the value (next "/" or end of string)
	endIdx := strings.Index(resource[startIdx:], "/")
	if endIdx < 0 {
		// Segment is at the end, return the value and prefix as remaining
		value = resource[startIdx:]
		if prefixLen > 0 {
			remaining = resource[:prefixLen]
		}
		return value, remaining
	}

	value = resource[startIdx : startIdx+endIdx]
	// Build remaining: prefix + suffix after the segment
	suffix := resource[startIdx+endIdx+1:]
	if prefixLen > 0 {
		remaining = resource[:prefixLen] + "/" + suffix
	} else {
		remaining = suffix
	}
	return value, remaining
}
